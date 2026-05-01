// vbas is the vb-autosuggest CLI (https://github.com/vinbh/vbas).
//
// Usage:
//
//	vbas complete --buffer "<line>" [--cursor N] [--specs DIR] [--json]
//	vbas version
//
// The shell hook in shell/zsh/vbas.zsh invokes `vbas complete` on Tab.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vinbh/vbas/internal/spec"
)

const version = "0.0.1"

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "complete":
		runComplete(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("vbas %s\n", version)
	case "help", "--help", "-h":
		usage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "vbas: unknown subcommand %q\n", os.Args[1])
		usage(os.Stderr)
		os.Exit(2)
	}
}

func usage(w *os.File) {
	fmt.Fprintln(w, `usage:
  vbas complete --buffer <line> [--cursor N] [--specs DIR] [--json]
  vbas version`)
}

func runComplete(args []string) {
	fs := flag.NewFlagSet("complete", flag.ExitOnError)
	buffer := fs.String("buffer", "", "the current command line")
	_ = fs.Int("cursor", -1, "cursor position (reserved; not yet used)")
	specsDir := fs.String("specs", defaultSpecsDir(), "directory containing JSON specs")
	asJSON := fs.Bool("json", false, "emit one JSON object per line instead of plain values")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *buffer == "" {
		return
	}

	tokens := strings.Fields(*buffer)
	if len(tokens) == 0 {
		return
	}
	cmd := tokens[0]

	loader := spec.NewLoader(*specsDir)
	s, err := loader.Load(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vbas: %v\n", err)
		os.Exit(1)
	}
	if s == nil {
		return
	}

	suggestions := spec.Match(s, *buffer)
	enc := json.NewEncoder(os.Stdout)
	for _, sug := range suggestions {
		if *asJSON {
			_ = enc.Encode(sug)
		} else {
			fmt.Println(sug.Value)
		}
	}
}

func defaultSpecsDir() string {
	if d := os.Getenv("VBAS_SPECS_DIR"); d != "" {
		return d
	}
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, ".config", "vbas", "specs")
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			return p
		}
	}
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "..", "specs")
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			return p
		}
	}
	return "specs"
}
