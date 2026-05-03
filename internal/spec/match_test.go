package spec

import "testing"

func gitSpec() *Spec {
	return &Spec{
		Name: "git",
		Subcommands: []Subcommand{
			{Name: Names{"checkout"}, Description: "Switch branches"},
			{Name: Names{"commit"}, Description: "Record changes",
				Options: []Option{
					{Name: Names{"-m", "--message"}, Description: "Commit message"},
					{Name: Names{"--amend"}, Description: "Amend previous commit"},
				}},
			{Name: Names{"clone"}, Description: "Clone a repo"},
		},
		Options: []Option{
			{Name: Names{"-v", "--version"}},
		},
	}
}

func TestMatchPrefixOneMatch(t *testing.T) {
	got := Match(gitSpec(), "git ch", "")
	if len(got) != 1 || got[0].Value != "checkout" {
		t.Fatalf("want [checkout], got %v", got)
	}
}

func TestMatchPrefixMultipleMatches(t *testing.T) {
	got := Match(gitSpec(), "git c", "")
	if len(got) != 3 {
		t.Fatalf("want 3 (checkout, commit, clone), got %d: %v", len(got), got)
	}
}

func TestMatchEmptyAfterSpace(t *testing.T) {
	got := Match(gitSpec(), "git ", "")
	// 3 subcommands + 1 top-level option (-v/--version emits 2 suggestions for both forms)
	if len(got) != 5 {
		t.Fatalf("want 5, got %d: %v", len(got), got)
	}
}

func TestMatchSubcommandOptions(t *testing.T) {
	got := Match(gitSpec(), "git commit -", "")
	if len(got) == 0 {
		t.Fatalf("want commit options, got none")
	}
	for _, s := range got {
		if s.Kind != "option" {
			t.Fatalf("want kind=option, got %q for %v", s.Kind, s)
		}
	}
}

func TestMatchUnknownCommand(t *testing.T) {
	if got := Match(gitSpec(), "notgit foo", ""); got != nil {
		t.Fatalf("want nil for unknown command, got %v", got)
	}
}

func TestMatchNoSpec(t *testing.T) {
	if got := Match(nil, "git ch", ""); got != nil {
		t.Fatalf("want nil for nil spec, got %v", got)
	}
}

func TestMatchDispatchesFolderGenerator(t *testing.T) {
	tmp := makeTree(t)
	cdSpec := &Spec{
		Name: "cd",
		Args: Args{{Name: "dir", Template: Templates{"folders"}}},
	}
	got := Match(cdSpec, "cd ", tmp)
	gotVals := values(got)
	if !equal(gotVals, []string{"sub/"}) {
		t.Fatalf("want [sub/], got %v", gotVals)
	}
}

func TestMatchDispatchesFilepathsWithPrefix(t *testing.T) {
	tmp := makeTree(t)
	catSpec := &Spec{
		Name: "cat",
		Args: Args{{Name: "file", Template: Templates{"filepaths"}, IsVariadic: true}},
	}
	got := Match(catSpec, "cat al", tmp)
	gotVals := values(got)
	if !equal(gotVals, []string{"alpha.txt"}) {
		t.Fatalf("want [alpha.txt], got %v", gotVals)
	}
}

func TestMatchVariadicCarriesAcrossPositions(t *testing.T) {
	tmp := makeTree(t)
	catSpec := &Spec{
		Name: "cat",
		Args: Args{{Name: "file", Template: Templates{"filepaths"}, IsVariadic: true}},
	}
	// After consuming one positional arg, generator should still fire on the next.
	got := Match(catSpec, "cat alpha.txt be", tmp)
	gotVals := values(got)
	if !equal(gotVals, []string{"beta.txt"}) {
		t.Fatalf("want [beta.txt], got %v", gotVals)
	}
}

func TestMatchSubcommandArgsOverrideRoot(t *testing.T) {
	tmp := makeTree(t)
	// Spec where `git checkout <file>` has its own filepath template, but `git`
	// itself does not — the matcher must use the subcommand's args after descent.
	gitWithCheckoutFiles := &Spec{
		Name: "git",
		Subcommands: []Subcommand{
			{
				Name: Names{"checkout"},
				Args: Args{{Name: "file", Template: Templates{"filepaths"}}},
			},
		},
	}
	got := Match(gitWithCheckoutFiles, "git checkout al", tmp)
	gotVals := values(got)
	if !equal(gotVals, []string{"alpha.txt"}) {
		t.Fatalf("want [alpha.txt], got %v", gotVals)
	}
}
