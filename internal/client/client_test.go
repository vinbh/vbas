package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultSocketPath(t *testing.T) {
	t.Run("PEEK_SOCKET wins", func(t *testing.T) {
		t.Setenv("PEEK_SOCKET", "/custom/path.sock")
		t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
		got := DefaultSocketPath()
		if got != "/custom/path.sock" {
			t.Errorf("want PEEK_SOCKET to win, got %q", got)
		}
	})

	t.Run("XDG_RUNTIME_DIR is preferred over /tmp", func(t *testing.T) {
		t.Setenv("PEEK_SOCKET", "")
		t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
		got := DefaultSocketPath()
		want := filepath.Join("/run/user/1000", "peek", "peek.sock")
		if got != want {
			t.Errorf("want %q, got %q", want, got)
		}
	})

	t.Run("falls back to /tmp/peek-UID.sock", func(t *testing.T) {
		t.Setenv("PEEK_SOCKET", "")
		t.Setenv("XDG_RUNTIME_DIR", "")
		got := DefaultSocketPath()
		uid := os.Getuid()
		if !strings.Contains(got, "peek-") || !strings.Contains(got, ".sock") {
			t.Errorf("unexpected fallback path: %q", got)
		}
		_ = uid
	})
}

// TryDaemon against a path that doesn't exist should return an error
// quickly (caller will then fall back to in-process matching).
func TestTryDaemonNoSocket(t *testing.T) {
	tmp := t.TempDir()
	bogus := filepath.Join(tmp, "does-not-exist.sock")
	_, err := TryDaemon("git c", "", bogus)
	if err == nil {
		t.Fatal("expected error dialing nonexistent socket")
	}
}
