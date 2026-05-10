// vbas is the vb-autosuggest CLI (https://github.com/vinbh/vbas).
//
// Usage:
//
//	vbas complete --buffer "<line>" [--cursor N] [--specs DIR] [--json] [--interactive]
//	vbas daemon   [--socket PATH] [--specs DIR]
//	vbas version
//
// In --interactive mode the dropdown UI is drawn on /dev/tty and the
// chosen value is written to stdout. Exit codes:
//
//	0  — picked a value (stdout has it) or user cancelled (stdout empty)
//	1  — internal error
//	2  — nothing to suggest (no spec, no matches); shell should fall through
//
// The complete subcommand transparently uses a long-running vbas daemon
// over a Unix socket when one is available (sub-millisecond round-trip).
// If no daemon is running, the request is answered in-process and a
// detached daemon is spawned in the background for the next request.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vinbh/vbas/internal/client"
	"github.com/vinbh/vbas/internal/daemon"
	"github.com/vinbh/vbas/internal/spec"
	"github.com/vinbh/vbas/internal/ui"
)

const version = "0.2.0"

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "complete":
		runComplete(os.Args[2:])
	case "daemon":
		runDaemon(os.Args[2:])
	case "setup":
		runSetup()
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
  vbas complete --buffer <line> [--cursor N] [--specs DIR] [--json] [--interactive]
  vbas daemon   [--socket PATH] [--specs DIR]
  vbas setup
  vbas version`)
}

func runComplete(args []string) {
	fs := flag.NewFlagSet("complete", flag.ExitOnError)
	buffer := fs.String("buffer", "", "the current command line")
	_ = fs.Int("cursor", -1, "cursor position (reserved; not yet used)")
	specsDir := fs.String("specs", defaultSpecsDir(), "directory containing JSON specs")
	asJSON := fs.Bool("json", false, "emit one JSON object per line instead of plain values")
	interactive := fs.Bool("interactive", false, "show dropdown UI; exit 2 when no spec/no matches")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	suggestions, err := getSuggestions(*buffer, *specsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vbas: %v\n", err)
		os.Exit(1)
	}

	if *interactive {
		outputInteractive(suggestions)
		return
	}
	outputPlain(suggestions, *asJSON)
}

func runDaemon(args []string) {
	fs := flag.NewFlagSet("daemon", flag.ExitOnError)
	sockPath := fs.String("socket", client.DefaultSocketPath(), "Unix socket path")
	specsDir := fs.String("specs", defaultSpecsDir(), "directory containing JSON specs")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if err := daemon.Run(context.Background(), *sockPath, *specsDir); err != nil {
		fmt.Fprintf(os.Stderr, "vbas-daemon: %v\n", err)
		os.Exit(1)
	}
}

// getSuggestions tries the daemon first; on failure, answers in-process
// AND fires off a detached daemon spawn so the next request hits the
// fast path. The fallback keeps vbas usable even when the daemon can't
// start (locked filesystem, weird permissions, etc).
func getSuggestions(buffer, specsDir string) ([]spec.Suggestion, error) {
	if buffer == "" {
		return nil, nil
	}

	cwd, _ := os.Getwd()
	sockPath := client.DefaultSocketPath()
	if s, err := client.TryDaemon(buffer, cwd, sockPath); err == nil {
		return s, nil
	}

	// Daemon unavailable. Spawn one for next time (best-effort).
	_ = client.SpawnDaemon(sockPath, specsDir)

	// Answer this request in-process so the user isn't kept waiting.
	return inProcessMatch(buffer, cwd, specsDir)
}

func inProcessMatch(buffer, cwd, specsDir string) ([]spec.Suggestion, error) {
	tokens := strings.Fields(buffer)
	if len(tokens) == 0 {
		return nil, nil
	}
	loader := spec.NewLoader(specsDir)
	s, err := loader.Load(tokens[0])
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	return spec.Match(s, buffer, cwd), nil
}

// outputInteractive handles the --interactive case. Exits 2 with empty
// stdout when there's nothing to offer (so the shell falls through);
// exits 0 with the picked value (or empty if cancelled) otherwise.
func outputInteractive(suggestions []spec.Suggestion) {
	if len(suggestions) == 0 {
		os.Exit(2)
	}
	if len(suggestions) == 1 {
		fmt.Println(suggestions[0].Value)
		return
	}
	items := make([]ui.Item, len(suggestions))
	for i, s := range suggestions {
		items[i] = ui.Item{Value: s.Value, Description: s.Description}
	}
	pick, err := ui.Run(items)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vbas: %v\n", err)
		os.Exit(1)
	}
	if pick != "" {
		fmt.Println(pick)
	}
}

func outputPlain(suggestions []spec.Suggestion, asJSON bool) {
	enc := json.NewEncoder(os.Stdout)
	for _, sug := range suggestions {
		if asJSON {
			_ = enc.Encode(sug)
		} else {
			fmt.Println(sug.Value)
		}
	}
}

func isDirExist(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
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
		exeDir := filepath.Dir(exe)
		// Development / release-tarball layout: binary sits next to specs/
		if p := filepath.Join(exeDir, "..", "specs"); isDirExist(p) {
			return filepath.Clean(p)
		}
		// Homebrew layout: /opt/homebrew/bin/vbas → /opt/homebrew/share/vbas/specs
		if p := filepath.Join(exeDir, "..", "share", "vbas", "specs"); isDirExist(p) {
			return filepath.Clean(p)
		}
	}
	return "specs"
}
