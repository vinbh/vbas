package ui

import "testing"

func TestTruncate(t *testing.T) {
	cases := []struct {
		in   string
		w    int
		want string
	}{
		{"", 5, ""},
		{"abc", 5, "abc"},
		{"abcde", 5, "abcde"},
		{"abcdef", 5, "abcd…"},
		{"a long string", 6, "a lon…"},
		{"emoji 🚀 test", 8, "emoji 🚀…"},
		{"anything", 0, ""},
		{"anything", -1, ""},
	}
	for _, c := range cases {
		got := truncate(c.in, c.w)
		if got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.in, c.w, got, c.want)
		}
	}
}

func TestAdjustViewport(t *testing.T) {
	cases := []struct {
		name                  string
		selected, view, rows  int
		want                  int
	}{
		{"selection within view", 3, 0, 10, 0},
		{"selection at top of view", 5, 5, 5, 5},
		{"selection scrolled below view", 10, 0, 10, 1},
		{"selection scrolled above view", 2, 5, 5, 2},
		{"selection at bottom edge", 4, 0, 5, 0},
		{"selection one past bottom edge", 5, 0, 5, 1},
	}
	for _, c := range cases {
		got := adjustViewport(c.selected, c.view, c.rows)
		if got != c.want {
			t.Errorf("%s: adjustViewport(sel=%d, view=%d, rows=%d) = %d, want %d",
				c.name, c.selected, c.view, c.rows, got, c.want)
		}
	}
}

func TestFormatRowEmptyDesc(t *testing.T) {
	got := formatRow(Item{Value: "checkout"}, false)
	if got == "" {
		t.Fatalf("formatRow returned empty string")
	}
	// Should contain the value without any description-color sequence.
	if contains(got, csiFaint) {
		t.Errorf("formatRow with empty desc should not include faint sequence: %q", got)
	}
}

func TestFormatRowWithDesc(t *testing.T) {
	got := formatRow(Item{Value: "commit", Description: "Record changes"}, false)
	if !contains(got, "commit") || !contains(got, "Record changes") {
		t.Errorf("formatRow missing value or description: %q", got)
	}
	if !contains(got, csiFaint) {
		t.Errorf("formatRow with desc should include faint sequence for description: %q", got)
	}
}

func TestFormatRowSelectedSkipsFaint(t *testing.T) {
	// Selected rows are wrapped in reverse video by the caller, so the
	// description column shouldn't add its own dim attribute.
	got := formatRow(Item{Value: "commit", Description: "Record changes"}, true)
	if contains(got, csiFaint) {
		t.Errorf("selected row should not apply faint to description: %q", got)
	}
}

func TestParseEscape(t *testing.T) {
	cases := []struct {
		name         string
		prefix, code byte
		want         key
	}{
		{"CSI Up", '[', 'A', keyUp},
		{"CSI Down", '[', 'B', keyDown},
		{"CSI Right", '[', 'C', keyRight},
		{"CSI Left", '[', 'D', keyLeft},
		{"SS3 Up (zsh ZLE app cursor mode)", 'O', 'A', keyUp},
		{"SS3 Down", 'O', 'B', keyDown},
		{"SS3 Right", 'O', 'C', keyRight},
		{"SS3 Left", 'O', 'D', keyLeft},
		{"unknown prefix → bare Esc", 'X', 'A', keyEsc},
		{"unknown CSI code", '[', 'Z', keyUnknown},
	}
	for _, c := range cases {
		got := parseEscape(c.prefix, c.code)
		if got != c.want {
			t.Errorf("%s: parseEscape(%q, %q) = %v, want %v",
				c.name, c.prefix, c.code, got, c.want)
		}
	}
}

func contains(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
