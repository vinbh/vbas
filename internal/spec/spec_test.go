package spec

import (
	"encoding/json"
	"testing"
)

func TestArgsUnmarshalSingle(t *testing.T) {
	var a Args
	if err := json.Unmarshal([]byte(`{"name":"file","description":"target"}`), &a); err != nil {
		t.Fatal(err)
	}
	if len(a) != 1 || a[0].Name != "file" || a[0].Description != "target" {
		t.Fatalf("want [{file,target}], got %+v", a)
	}
}

func TestArgsUnmarshalArray(t *testing.T) {
	var a Args
	if err := json.Unmarshal([]byte(`[{"name":"src"},{"name":"dst"}]`), &a); err != nil {
		t.Fatal(err)
	}
	if len(a) != 2 || a[0].Name != "src" || a[1].Name != "dst" {
		t.Fatalf("want [src,dst], got %+v", a)
	}
}

func TestArgsUnmarshalEmptyArray(t *testing.T) {
	var a Args
	if err := json.Unmarshal([]byte(`[]`), &a); err != nil {
		t.Fatal(err)
	}
	if len(a) != 0 {
		t.Fatalf("want empty, got %+v", a)
	}
}

func TestArgUnmarshalTemplateString(t *testing.T) {
	var a Arg
	if err := json.Unmarshal([]byte(`{"name":"path","template":"filepaths"}`), &a); err != nil {
		t.Fatal(err)
	}
	if len(a.Template) != 1 || a.Template[0] != "filepaths" {
		t.Fatalf("want [filepaths], got %+v", a.Template)
	}
}

func TestArgUnmarshalTemplateArray(t *testing.T) {
	var a Arg
	if err := json.Unmarshal([]byte(`{"name":"path","template":["filepaths","folders"]}`), &a); err != nil {
		t.Fatal(err)
	}
	if len(a.Template) != 2 || a.Template[0] != "filepaths" || a.Template[1] != "folders" {
		t.Fatalf("want [filepaths,folders], got %+v", a.Template)
	}
}

func TestArgUnmarshalIsVariadic(t *testing.T) {
	var a Arg
	if err := json.Unmarshal([]byte(`{"name":"src","isVariadic":true,"template":["filepaths","folders"]}`), &a); err != nil {
		t.Fatal(err)
	}
	if !a.IsVariadic {
		t.Fatalf("want isVariadic true, got %+v", a)
	}
}

// Inline a Fig-shaped option that uses the single-object form for `args`
// (e.g. `git commit --message <msg>`) and verify it parses end-to-end.
func TestSpecParsesFigStyleSingleArg(t *testing.T) {
	raw := `{
  "name": "commit",
  "options": [
    {"name": ["-m","--message"], "args": {"name": "msg"}}
  ]
}`
	var s Subcommand
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatal(err)
	}
	if len(s.Options) != 1 || len(s.Options[0].Args) != 1 ||
		s.Options[0].Args[0].Name != "msg" {
		t.Fatalf("want -m --message with single arg msg, got %+v", s.Options[0])
	}
}
