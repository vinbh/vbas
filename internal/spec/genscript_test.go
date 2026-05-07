package spec

import (
	"testing"
)

func TestParseScriptOutput(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		token  string
		wantV  []string // expected Value fields, in order
		wantD  []string // expected Description fields, in order (parallel to wantV)
	}{
		{
			name:  "git branch: strips current-branch marker",
			raw:   "* main\n  feature/foo\n  feature/bar\n",
			token: "",
			wantV: []string{"main", "feature/foo", "feature/bar"},
			wantD: []string{"", "", ""},
		},
		{
			name:  "git branch: prefix filter",
			raw:   "* main\n  feature/foo\n  feature/bar\n",
			token: "feature",
			wantV: []string{"feature/foo", "feature/bar"},
			wantD: []string{"", ""},
		},
		{
			name:  "git log --oneline: hash+description",
			raw:   "abc1234 commit message here\ndef5678 another commit\n",
			token: "",
			wantV: []string{"abc1234", "def5678"},
			wantD: []string{"commit message here", "another commit"},
		},
		{
			name:  "git status --short: strips 2-char status code",
			raw:   " M README.md\n?? new.txt\nM  staged.go\n",
			token: "",
			wantV: []string{"README.md", "new.txt", "staged.go"},
			wantD: []string{"", "", ""},
		},
		{
			name:  "tab-separated value+description",
			raw:   "origin\tgit@github.com:user/repo.git (fetch)\nupstream\tgit@github.com:org/repo.git (fetch)\n",
			token: "",
			wantV: []string{"origin", "upstream"},
			wantD: []string{"git@github.com:user/repo.git (fetch)", "git@github.com:org/repo.git (fetch)"},
		},
		{
			name:  "plain list: one value per line",
			raw:   "my-container\nanother-container\n",
			token: "",
			wantV: []string{"my-container", "another-container"},
			wantD: []string{"", ""},
		},
		{
			name:  "empty lines are skipped",
			raw:   "\nfoo\n\nbar\n\n",
			token: "",
			wantV: []string{"foo", "bar"},
			wantD: []string{"", ""},
		},
		{
			name:  "token filter on plain list",
			raw:   "main\ndevelop\nfeature/x\n",
			token: "feat",
			wantV: []string{"feature/x"},
			wantD: []string{""},
		},
		{
			name:  "empty input returns nil",
			raw:   "",
			token: "",
			wantV: nil,
			wantD: nil,
		},
		{
			name:  "whitespace-only lines are skipped",
			raw:   "   \n\t\nfoo\n",
			token: "",
			wantV: []string{"foo"},
			wantD: []string{""},
		},
		{
			name:  "crlf line endings handled",
			raw:   "foo\r\nbar\r\n",
			token: "",
			wantV: []string{"foo", "bar"},
			wantD: []string{"", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseScriptOutput(tt.raw, tt.token)
			if len(got) != len(tt.wantV) {
				t.Fatalf("got %d suggestions, want %d\ngot: %v", len(got), len(tt.wantV), got)
			}
			for i, s := range got {
				if s.Value != tt.wantV[i] {
					t.Errorf("[%d] Value = %q, want %q", i, s.Value, tt.wantV[i])
				}
				if s.Description != tt.wantD[i] {
					t.Errorf("[%d] Description = %q, want %q", i, s.Description, tt.wantD[i])
				}
				if s.Kind != "arg" {
					t.Errorf("[%d] Kind = %q, want %q", i, s.Kind, "arg")
				}
			}
		})
	}
}

func TestRunScript(t *testing.T) {
	// Use a real command to verify the full pipeline.
	got := runScript([]string{"printf", "main\nfeature/foo\n"}, "", "")
	if len(got) != 2 {
		t.Fatalf("want 2 suggestions, got %d: %v", len(got), got)
	}
	if got[0].Value != "main" {
		t.Errorf("got[0].Value = %q, want %q", got[0].Value, "main")
	}
	if got[1].Value != "feature/foo" {
		t.Errorf("got[1].Value = %q, want %q", got[1].Value, "feature/foo")
	}
}

func TestRunScriptMissingBinary(t *testing.T) {
	// A missing binary should return no suggestions, not panic.
	got := runScript([]string{"/nonexistent/binary-that-does-not-exist"}, "", "")
	if len(got) != 0 {
		t.Errorf("expected no suggestions for missing binary, got %v", got)
	}
}

func TestRunGeneratorsDeduplication(t *testing.T) {
	gens := Generators{
		{Script: []string{"printf", "main\nfeature\n"}},
		{Script: []string{"printf", "main\ndevelop\n"}},
	}
	got := RunGenerators(gens, "", "")
	seen := map[string]int{}
	for _, s := range got {
		seen[s.Value]++
	}
	if seen["main"] != 1 {
		t.Errorf("'main' appeared %d times, want exactly 1 (deduplication failed)", seen["main"])
	}
	if seen["feature"] != 1 || seen["develop"] != 1 {
		t.Errorf("unexpected results: %v", got)
	}
}
