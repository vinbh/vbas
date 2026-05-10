package spec

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
	"unicode"
)

const scriptTimeout = 5 * time.Second

// RunGenerators merges results from all generators in gens. Script generators
// execute a shell command and parse stdout; template generators delegate to
// RunTemplates. Duplicate values (by Value field) are deduplicated.
//
// Specs in ~/.config/peek/specs/ are user-controlled; imported Fig specs come
// from withfig/autocomplete (MIT). Scripts run with the user's shell cwd and
// inherit no special privileges beyond the daemon process itself.
func RunGenerators(gens Generators, cwd, token string) []Suggestion {
	var out []Suggestion
	seen := make(map[string]bool)
	add := func(sug []Suggestion) {
		for _, s := range sug {
			if !seen[s.Value] {
				seen[s.Value] = true
				out = append(out, s)
			}
		}
	}
	for _, gen := range gens {
		if len(gen.Script) > 0 {
			add(runScript(gen.Script, cwd, token))
		}
		if len(gen.Template) > 0 {
			add(RunTemplates(gen.Template, cwd, token))
		}
	}
	return out
}

func runScript(script []string, cwd, token string) []Suggestion {
	ctx, cancel := context.WithTimeout(context.Background(), scriptTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, script[0], script[1:]...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	_ = cmd.Run() // partial output on error is still useful
	return parseScriptOutput(buf.String(), token)
}

// parseScriptOutput converts raw stdout from a generator script into
// Suggestion values. It handles the common Fig output conventions:
//
//   - git branch:     "* main" / "  feature" — strips "* " prefix
//   - git status -s:  " M file.go" / "?? new" — strips 2-char status code
//   - oneline log:    "abc1234 commit message" — hash=value, rest=description
//   - tab-separated:  "value\tdescription"
//   - plain list:     one value per line
func parseScriptOutput(raw, token string) []Suggestion {
	var out []Suggestion
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")

		// git branch current-branch marker ("* main" → "main")
		if strings.HasPrefix(line, "* ") {
			line = line[2:]
		}

		// git status short format: 2-char status code + space (e.g. " M ", "?? ")
		if len(line) >= 3 && line[2] == ' ' && isStatusPrefix(line[:2]) {
			line = line[3:]
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var value, description string
		if idx := strings.IndexByte(line, '\t'); idx >= 0 {
			value = strings.TrimSpace(line[:idx])
			description = strings.TrimSpace(line[idx+1:])
		} else if idx := strings.IndexByte(line, ' '); idx >= 0 {
			value = line[:idx]
			description = strings.TrimSpace(line[idx+1:])
		} else {
			value = line
		}

		if value == "" {
			continue
		}
		if token != "" && !strings.HasPrefix(value, token) {
			continue
		}
		out = append(out, Suggestion{Value: value, Description: description, Kind: "arg"})
	}
	return out
}

// isStatusPrefix reports whether s is a 2-char git status code (uppercase
// letters, '?', '!', or space — the full set git uses).
func isStatusPrefix(s string) bool {
	for _, c := range s {
		if !unicode.IsUpper(c) && c != '?' && c != '!' && c != ' ' {
			return false
		}
	}
	return true
}
