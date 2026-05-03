package spec

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// makeTree creates a small fs layout under tmp:
//
//	tmp/
//	  alpha.txt
//	  beta.txt
//	  README
//	  .hidden
//	  sub/
//	    inner.go
//	    nested/
func makeTree(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	for _, f := range []string{"alpha.txt", "beta.txt", "README", ".hidden"} {
		if err := os.WriteFile(filepath.Join(tmp, f), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(tmp, "sub", "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "sub", "inner.go"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	return tmp
}

func values(s []Suggestion) []string {
	out := make([]string, len(s))
	for i, x := range s {
		out[i] = x.Value
	}
	sort.Strings(out)
	return out
}

func TestFilepathsListsCwdNoFilter(t *testing.T) {
	tmp := makeTree(t)
	got := RunTemplates(Templates{"filepaths"}, tmp, "")
	want := []string{"README", "alpha.txt", "beta.txt", "sub/"}
	if g := values(got); !equal(g, want) {
		t.Fatalf("want %v, got %v", want, g)
	}
}

func TestFoldersFiltersOutFiles(t *testing.T) {
	tmp := makeTree(t)
	got := RunTemplates(Templates{"folders"}, tmp, "")
	if g := values(got); !equal(g, []string{"sub/"}) {
		t.Fatalf("want [sub/], got %v", g)
	}
}

func TestFilepathsBareNamePrefixFilters(t *testing.T) {
	tmp := makeTree(t)
	got := RunTemplates(Templates{"filepaths"}, tmp, "al")
	if g := values(got); !equal(g, []string{"alpha.txt"}) {
		t.Fatalf("want [alpha.txt], got %v", g)
	}
}

func TestFilepathsDescendsIntoSubdir(t *testing.T) {
	tmp := makeTree(t)
	got := RunTemplates(Templates{"filepaths"}, tmp, "sub/")
	if g := values(got); !equal(g, []string{"sub/inner.go", "sub/nested/"}) {
		t.Fatalf("want sub/inner.go and sub/nested/, got %v", g)
	}
}

func TestFilepathsFiltersBasenameInsideSubdir(t *testing.T) {
	tmp := makeTree(t)
	got := RunTemplates(Templates{"filepaths"}, tmp, "sub/in")
	if g := values(got); !equal(g, []string{"sub/inner.go"}) {
		t.Fatalf("want [sub/inner.go], got %v", g)
	}
}

func TestFilepathsAbsolutePath(t *testing.T) {
	tmp := makeTree(t)
	got := RunTemplates(Templates{"filepaths"}, "/some/where/else", tmp+"/")
	want := []string{
		filepath.Join(tmp, "README"),
		filepath.Join(tmp, "alpha.txt"),
		filepath.Join(tmp, "beta.txt"),
		filepath.Join(tmp, "sub") + "/",
	}
	// Suggestions should each begin with `tmp+"/"`.
	for _, s := range got {
		if !strings.HasPrefix(s.Value, tmp+"/") {
			t.Fatalf("expected absolute prefix %q, got %q", tmp+"/", s.Value)
		}
	}
	if g := values(got); !equal(g, sortedCopy(want)) {
		t.Fatalf("want %v, got %v", sortedCopy(want), g)
	}
}

func TestFilepathsHiddenOnlyWhenDotPrefix(t *testing.T) {
	tmp := makeTree(t)
	if got := RunTemplates(Templates{"filepaths"}, tmp, ""); contains(values(got), ".hidden") {
		t.Fatalf(".hidden should be skipped without dot prefix, got %v", values(got))
	}
	got := RunTemplates(Templates{"filepaths"}, tmp, ".")
	if g := values(got); !equal(g, []string{".hidden"}) {
		t.Fatalf("dot prefix should reveal hidden entries, got %v", g)
	}
}

func TestFilepathsTildeExpansionPreservesTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, "myfile"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	got := RunTemplates(Templates{"filepaths"}, "/somewhere/unrelated", "~/")
	if g := values(got); !equal(g, []string{"~/myfile"}) {
		t.Fatalf("want [~/myfile] (tilde preserved), got %v", g)
	}
}

func TestUnknownTemplateReturnsNil(t *testing.T) {
	tmp := makeTree(t)
	if got := RunTemplates(Templates{"history"}, tmp, ""); got != nil {
		t.Fatalf("history template is not implemented yet, want nil got %v", got)
	}
	if got := RunTemplates(nil, tmp, ""); got != nil {
		t.Fatalf("nil templates should return nil, got %v", got)
	}
}

func TestFilepathsNonexistentDir(t *testing.T) {
	if got := RunTemplates(Templates{"filepaths"}, "/this/path/does/not/exist", ""); got != nil {
		t.Fatalf("nonexistent dir should return nil, got %v", got)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sortedCopy(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

func contains(s []string, want string) bool {
	for _, x := range s {
		if x == want {
			return true
		}
	}
	return false
}
