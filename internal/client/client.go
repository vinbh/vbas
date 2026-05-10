// Package client lets the peek CLI talk to a running peek-daemon over
// a Unix socket, with auto-spawn of a detached daemon if one isn't
// running yet. Callers are expected to fall back to in-process matching
// if the client returns an error — that fallback is what keeps peek
// working even when the daemon can't start (locked filesystem,
// permission issue, whatever).
package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/vinbh/peek/internal/proto"
	"github.com/vinbh/peek/internal/spec"
)

// connectTimeout is the deadline for the initial dial. Kept short so that
// "daemon not running" fails fast and the client falls back to in-process.
const connectTimeout = 200 * time.Millisecond

// responseTimeout is the deadline for reading the daemon's reply once
// connected. Generators may run external commands (git branch, kubectl get
// pods, …) — 6 s covers the script timeout (5 s) plus encoding overhead.
const responseTimeout = 6 * time.Second

// DefaultSocketPath returns the canonical Unix socket path for this user.
//
// Order: $PEEK_SOCKET, $XDG_RUNTIME_DIR/peek/peek.sock, /tmp/peek-$UID.sock.
// $XDG_RUNTIME_DIR is the right place on systemd Linux (tmpfs, cleared
// on logout); the /tmp fallback is for systems that don't set it.
func DefaultSocketPath() string {
	if p := os.Getenv("PEEK_SOCKET"); p != "" {
		return p
	}
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "peek", "peek.sock")
	}
	return filepath.Join(os.TempDir(), "peek-"+strconv.Itoa(os.Getuid())+".sock")
}

// TryDaemon makes a single attempt to fetch suggestions from the daemon.
// Returns (suggestions, nil) on success; (nil, err) if the daemon isn't
// reachable or the request failed. Does NOT spawn a daemon — that's a
// separate decision for the caller via SpawnDaemon.
func TryDaemon(buffer, cwd, sockPath string) ([]spec.Suggestion, error) {
	conn, err := net.DialTimeout("unix", sockPath, connectTimeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(responseTimeout))

	if err := json.NewEncoder(conn).Encode(proto.Request{
		Op:     "complete",
		Buffer: buffer,
		Cwd:    cwd,
	}); err != nil {
		return nil, err
	}

	var resp proto.Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return resp.Suggestions, nil
}

// SpawnDaemon fires off a detached `peek daemon` subprocess. Returns
// once the subprocess has been started — does NOT wait for it to bind
// the socket. The caller should answer the current request via the
// in-process fallback; the daemon will be ready for the next one.
//
// On Linux we use Setsid so the daemon survives the parent exiting.
// stderr goes to a log file alongside the socket so `cat $XDG_RUNTIME_DIR/
// peek/peek-daemon.log` is the debug entry point.
func SpawnDaemon(sockPath, specsDir string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate self: %w", err)
	}

	logDir := filepath.Dir(sockPath)
	_ = os.MkdirAll(logDir, 0700)
	logPath := filepath.Join(logDir, "peek-daemon.log")
	logFile, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)

	args := []string{"daemon", "--socket", sockPath}
	if specsDir != "" {
		args = append(args, "--specs", specsDir)
	}
	cmd := exec.Command(exe, args...)
	cmd.Stdin = nil
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawn daemon: %w", err)
	}
	// Detach — don't wait, don't keep the *Process around in our PCB.
	_ = cmd.Process.Release()
	return nil
}
