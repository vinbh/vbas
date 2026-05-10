package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	peekassets "github.com/vinbh/peek"
)

// runSetup implements "peek setup": extracts embedded specs and shell hooks
// to ~/.config/peek/ and offers to wire them into detected shell rc files.
// This is the go-install path — no separate download needed after:
//
//	go install github.com/vinbh/peek/cmd/peek@latest
//	peek setup
func runSetup() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "peek setup: %v\n", err)
		os.Exit(1)
	}
	configDir := filepath.Join(home, ".config", "peek")

	// 1. Extract specs.
	fmt.Println("==> extracting specs")
	specsDir := filepath.Join(configDir, "specs")
	if err := extractDir(peekassets.EmbeddedSpecs, "specs", specsDir); err != nil {
		fmt.Fprintf(os.Stderr, "peek setup: extract specs: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("    %s\n", specsDir)

	// 2. Extract shell hooks.
	fmt.Println("==> extracting shell hooks")
	type hook struct{ src, dst string }
	hooks := []hook{
		{"shell/zsh/peek.zsh", filepath.Join(configDir, "peek.zsh")},
		{"shell/bash/peek.bash", filepath.Join(configDir, "peek.bash")},
	}
	for _, h := range hooks {
		if err := extractSingleFile(peekassets.EmbeddedHooks, h.src, h.dst, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "peek setup: extract %s: %v\n", h.src, err)
			os.Exit(1)
		}
		fmt.Printf("    %s\n", h.dst)
	}

	// 3. Stop any running daemon so the next completion uses the fresh specs.
	_ = exec.Command("pkill", "-KILL", "-f", "peek daemon").Run()

	// 4. Detect shells and offer to wire rc files.
	fmt.Println("==> shell integration")
	type shell struct{ name, rcFile, hookFile string }
	shells := []shell{
		{"zsh", filepath.Join(home, ".zshrc"), filepath.Join(configDir, "peek.zsh")},
		{"bash", filepath.Join(home, ".bashrc"), filepath.Join(configDir, "peek.bash")},
	}
	wiredAny := false
	for _, s := range shells {
		if _, err := exec.LookPath(s.name); err != nil {
			continue
		}
		offerRC(s.name, s.rcFile, s.hookFile)
		wiredAny = true
	}
	if !wiredAny {
		fmt.Println("  (no supported shells detected)")
	}

	fmt.Println()
	fmt.Println("Done. Open a new shell or re-source your rc file to activate peek.")
}

// extractDir walks the embedded FS subtree at src and writes every file to
// the corresponding path under dst, creating directories as needed.
func extractDir(fsys fs.FS, src, dst string) error {
	return fs.WalkDir(fsys, src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func extractSingleFile(fsys fs.FS, src, dst string, mode fs.FileMode) error {
	data, err := fs.ReadFile(fsys, src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, mode)
}

// offerRC offers to append a source line to a shell's rc file.
// It reads the user's answer from /dev/tty so it works even when
// stdin is a pipe (e.g. curl | bash).
func offerRC(shell, rcFile, hookPath string) {
	srcLine := "source " + hookPath
	if fileContainsLine(rcFile, srcLine) {
		fmt.Printf("  %s: already enabled in %s\n", shell, rcFile)
		return
	}

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		// Non-interactive environment: print hint only.
		fmt.Printf("  %s: add to %s:\n      %s\n", shell, rcFile, srcLine)
		return
	}
	defer tty.Close()

	fmt.Fprintf(tty, "  Enable peek in %s (%s)? [Y/n] ", shell, filepath.Base(rcFile))
	sc := bufio.NewScanner(tty)
	sc.Scan()
	ans := strings.TrimSpace(sc.Text())

	if ans == "" || strings.EqualFold(ans[0:1], "y") {
		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  cannot write %s: %v\n", rcFile, err)
			return
		}
		defer f.Close()
		fmt.Fprintf(f, "\n# peek autosuggest\n%s\n", srcLine)
		fmt.Printf("  Added. Run: source %s\n", rcFile)
	} else {
		fmt.Printf("  Skipped. Add manually to %s:\n      %s\n", rcFile, srcLine)
	}
}

func fileContainsLine(path, substr string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}
