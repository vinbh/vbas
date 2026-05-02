package spec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoaderHandRolledWinsOverFig(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "fig"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "git.json"),
		[]byte(`{"name":"git","description":"hand-rolled"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "fig", "git.json"),
		[]byte(`{"name":"git","description":"fig"}`), 0600); err != nil {
		t.Fatal(err)
	}

	s, err := NewLoader(tmp).Load("git")
	if err != nil {
		t.Fatal(err)
	}
	if s == nil || s.Description != "hand-rolled" {
		t.Fatalf("want hand-rolled to win, got %+v", s)
	}
}

func TestLoaderFallsBackToFig(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "fig"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "fig", "docker.json"),
		[]byte(`{"name":"docker","description":"from fig"}`), 0600); err != nil {
		t.Fatal(err)
	}

	s, err := NewLoader(tmp).Load("docker")
	if err != nil {
		t.Fatal(err)
	}
	if s == nil || s.Name != "docker" || s.Description != "from fig" {
		t.Fatalf("want fig fallback, got %+v", s)
	}
}

func TestLoaderMissing(t *testing.T) {
	s, err := NewLoader(t.TempDir()).Load("nope")
	if err != nil {
		t.Fatal(err)
	}
	if s != nil {
		t.Fatalf("want nil for missing spec, got %+v", s)
	}
}

// Imported Fig specs prepend metadata keys (_license, _source, _generated).
// Make sure those don't trip the parser — Go's json decoder ignores unknown
// fields by default, but if anyone adds a strict-decode codepath they'll
// notice this test fails.
func TestLoaderIgnoresUnknownTopLevelKeys(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "x.json"), []byte(`{
  "_license": "MIT",
  "_source": "https://example/x.ts",
  "_generated": "tools/import-fig/import.ts",
  "name": "x",
  "subcommands": [{"name": "sub"}]
}`), 0600); err != nil {
		t.Fatal(err)
	}

	s, err := NewLoader(tmp).Load("x")
	if err != nil {
		t.Fatalf("want no error parsing fig-shaped JSON, got %v", err)
	}
	if s == nil || s.Name != "x" || len(s.Subcommands) != 1 {
		t.Fatalf("want parsed spec with 1 subcommand, got %+v", s)
	}
}
