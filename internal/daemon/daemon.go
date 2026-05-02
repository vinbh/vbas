// Package daemon implements the vbas long-running matching server.
//
// On startup it binds a Unix domain socket (with stale-socket recovery)
// and serves one Request → one Response per connection. Spec parsing
// is done once per command and cached in memory for the daemon's
// lifetime — this is the whole reason the daemon exists.
//
// The daemon does NOT render the dropdown UI. The client (which still
// has /dev/tty access) does that. The daemon's job is just: load spec,
// match buffer, return suggestions.
package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/vinbh/vbas/internal/proto"
	"github.com/vinbh/vbas/internal/spec"
)

// Run binds the Unix socket at sockPath and serves until ctx is cancelled
// or SIGTERM/SIGINT arrives. specsDir is the directory holding <cmd>.json
// files; it's loaded lazily as commands are seen.
func Run(ctx context.Context, sockPath, specsDir string) error {
	if err := os.MkdirAll(filepath.Dir(sockPath), 0700); err != nil {
		return fmt.Errorf("mkdir socket dir: %w", err)
	}

	if err := reclaimStaleSocket(sockPath); err != nil {
		return err
	}

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return fmt.Errorf("listen %s: %w", sockPath, err)
	}
	// Lock down to user-only — Unix permissions are our only auth.
	_ = os.Chmod(sockPath, 0600)
	defer os.Remove(sockPath)

	// Plumb context cancellation + signals to the listener.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigCh)

	go func() {
		select {
		case <-ctx.Done():
		case s := <-sigCh:
			log.Printf("vbas-daemon: received %s, shutting down", s)
		}
		_ = ln.Close()
	}()

	loader := spec.NewLoader(specsDir)
	var wg sync.WaitGroup

	log.Printf("vbas-daemon: listening on %s, specs=%s", sockPath, specsDir)
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				wg.Wait()
				return nil
			}
			log.Printf("vbas-daemon: accept: %v", err)
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			handleConn(conn, loader)
		}()
	}
}

// reclaimStaleSocket tests whether sockPath is held by a live daemon and
// either errors out (if alive) or unlinks the stale file (if dead).
func reclaimStaleSocket(sockPath string) error {
	if _, err := os.Stat(sockPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat socket: %w", err)
	}
	// File exists. Probe.
	conn, err := net.Dial("unix", sockPath)
	if err == nil {
		conn.Close()
		return fmt.Errorf("another vbas-daemon is already running on %s", sockPath)
	}
	// Not alive — unlink.
	return os.Remove(sockPath)
}

func handleConn(conn net.Conn, loader *spec.Loader) {
	defer conn.Close()

	var req proto.Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		writeError(conn, "invalid request: "+err.Error())
		return
	}

	switch req.Op {
	case "complete":
		_ = json.NewEncoder(conn).Encode(proto.Response{
			Suggestions: completeBuffer(req.Buffer, loader),
		})
	case "ping":
		_ = json.NewEncoder(conn).Encode(proto.Response{})
	default:
		writeError(conn, "unknown op: "+req.Op)
	}
}

func completeBuffer(buffer string, loader *spec.Loader) []spec.Suggestion {
	if buffer == "" {
		return nil
	}
	tokens := strings.Fields(buffer)
	if len(tokens) == 0 {
		return nil
	}
	s, err := loader.Load(tokens[0])
	if err != nil || s == nil {
		return nil
	}
	return spec.Match(s, buffer)
}

func writeError(conn net.Conn, msg string) {
	_ = json.NewEncoder(conn).Encode(proto.Response{Error: msg})
}
