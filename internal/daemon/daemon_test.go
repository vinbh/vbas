package daemon

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vinbh/peek/internal/proto"
)

// Spin up a daemon in a temp dir with a tiny git spec, send it a complete
// request over the socket, verify the suggestions come back.
func TestEndToEnd(t *testing.T) {
	tmp := t.TempDir()
	sockPath := filepath.Join(tmp, "peek.sock")
	specsDir := filepath.Join(tmp, "specs")
	if err := os.MkdirAll(specsDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "git.json"), []byte(`{
  "name": "git",
  "subcommands": [
    {"name": "checkout", "description": "Switch branches"},
    {"name": "commit",   "description": "Record changes"},
    {"name": "clone",    "description": "Clone a repo"}
  ]
}`), 0600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, sockPath, specsDir)
	}()

	// Wait for the socket to appear.
	if !waitForSocket(sockPath, 2*time.Second) {
		t.Fatalf("socket %s never appeared", sockPath)
	}

	// Send a complete request.
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(proto.Request{
		Op:     "complete",
		Buffer: "git c",
	}); err != nil {
		t.Fatal(err)
	}

	var resp proto.Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error != "" {
		t.Fatalf("daemon error: %s", resp.Error)
	}
	if len(resp.Suggestions) != 3 {
		t.Errorf("want 3 suggestions (checkout, commit, clone), got %d: %+v",
			len(resp.Suggestions), resp.Suggestions)
	}

	// Clean shutdown.
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("daemon exit error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("daemon did not shut down")
	}
}

func TestStaleSocketReclaim(t *testing.T) {
	tmp := t.TempDir()
	sockPath := filepath.Join(tmp, "peek.sock")
	// Touch the socket file so it exists but isn't bound.
	f, err := os.Create(sockPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	// Run should reclaim it and start successfully.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- Run(ctx, sockPath, "") }()

	if !waitForSocket(sockPath, 2*time.Second) {
		t.Fatal("daemon failed to bind after reclaiming stale socket")
	}
	cancel()
	<-done
}

func waitForSocket(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if conn, err := net.Dial("unix", path); err == nil {
			conn.Close()
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}
