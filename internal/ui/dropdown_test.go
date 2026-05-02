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

func TestRefilter(t *testing.T) {
	items := []Item{
		{Value: "checkout", Description: "Switch branches"},
		{Value: "commit", Description: "Record changes"},
		{Value: "clone", Description: "Clone a repository"},
		{Value: "stash", Description: "Stash changes"},
		{Value: "diff", Description: "Show changes"},
	}
	cases := []struct {
		name  string
		query string
		want  []string
	}{
		{"empty query matches all", "", []string{"checkout", "commit", "clone", "stash", "diff"}},
		{"value substring", "co", []string{"commit"}},
		{"value prefix", "clo", []string{"clone"}},
		{"description substring", "changes", []string{"commit", "stash", "diff"}},
		{"case insensitive", "CHECKOUT", []string{"checkout"}},
		{"no match", "xyz", nil},
	}
	for _, c := range cases {
		d := &dropdown{items: items, query: c.query}
		d.refilter()
		got := make([]string, 0, len(d.filtered))
		for _, idx := range d.filtered {
			got = append(got, d.items[idx].Value)
		}
		if !sliceEqual(got, c.want) {
			t.Errorf("%s (query=%q): got %v, want %v", c.name, c.query, got, c.want)
		}
	}
}

func TestRefilterResetsSelection(t *testing.T) {
	d := &dropdown{
		items:     []Item{{Value: "a"}, {Value: "b"}, {Value: "c"}},
		selected:  2,
		viewStart: 1,
	}
	d.query = "b"
	d.refilter()
	if d.selected != 0 || d.viewStart != 0 {
		t.Errorf("refilter should reset selected/viewStart to 0; got selected=%d viewStart=%d",
			d.selected, d.viewStart)
	}
}

func sliceEqual(a, b []string) bool {
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

func TestRefilterRanksPrefixThenSubstringThenDescription(t *testing.T) {
	// Simulate the "git c" scenario: many subcommands have a description
	// containing "c" but only a few start with "c". Prefix-name hits should
	// come first, value-substring next, description-only matches last.
	items := []Item{
		{Value: "add", Description: "Add file contents to the index"}, // desc-only
		{Value: "branch", Description: "List, create, or delete branches"},
		{Value: "checkout", Description: "Switch branches"}, // prefix
		{Value: "cherry-pick", Description: "Apply changes"}, // prefix
		{Value: "clean", Description: "Remove untracked files"}, // prefix
		{Value: "commit", Description: "Record changes"}, // prefix
		{Value: "diff", Description: "Show changes"},  // desc-only
		{Value: "switch", Description: "Switch branches"}, // value-substring
		{Value: "init", Description: "Create an empty repo"}, // desc-only
	}
	d := &dropdown{items: items, query: "c"}
	d.refilter()

	got := make([]string, 0, len(d.filtered))
	for _, idx := range d.filtered {
		got = append(got, items[idx].Value)
	}

	// Tier 1 (prefix): checkout, cherry-pick, clean, commit (in original order)
	// Tier 2 (value-substring): branch, switch
	// Tier 3 (description-only): add, diff, init
	want := []string{
		"checkout", "cherry-pick", "clean", "commit",
		"branch", "switch",
		"add", "diff", "init",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d items, want %d. got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("position %d: got %q, want %q. full got=%v", i, got[i], want[i], got)
		}
	}
}

func TestRefilterEmptyQueryShowsAllInOrder(t *testing.T) {
	items := []Item{
		{Value: "alpha"},
		{Value: "bravo"},
		{Value: "charlie"},
	}
	d := &dropdown{items: items, query: ""}
	d.refilter()

	if len(d.filtered) != 3 {
		t.Fatalf("want all 3 items, got %d", len(d.filtered))
	}
	for i, idx := range d.filtered {
		if idx != i {
			t.Errorf("expected original order, position %d has index %d", i, idx)
		}
	}
}

func TestRefilterCaseInsensitive(t *testing.T) {
	items := []Item{
		{Value: "ChEcKoUt"},
		{Value: "Switch"},
	}
	d := &dropdown{items: items, query: "CHECK"}
	d.refilter()

	if len(d.filtered) != 1 || d.filtered[0] != 0 {
		t.Fatalf("want 1 match (ChEcKoUt), got %v", d.filtered)
	}
}
