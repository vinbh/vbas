package spec

import "strings"

// Suggestion is a single completion candidate.
type Suggestion struct {
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Kind        string `json:"kind"` // "subcommand" | "option" | "arg"
}

// Match returns suggestions for the given buffer using the provided spec.
// M1 ignores the cursor and always completes the trailing token of the buffer.
// cwd is the client's working directory; used by filepath/folder generators
// (M5.5). Pass an empty string when not applicable — Match still works, but
// generators that need cwd will be skipped.
func Match(s *Spec, buffer, cwd string) []Suggestion {
	if s == nil {
		return nil
	}
	tokens := strings.Fields(buffer)
	if len(tokens) == 0 || tokens[0] != s.Name {
		return nil
	}

	endsWithSpace := strings.HasSuffix(buffer, " ")
	rest := tokens[1:]

	var prefix string
	if !endsWithSpace && len(rest) > 0 {
		prefix = rest[len(rest)-1]
		rest = rest[:len(rest)-1]
	}

	curSubs := s.Subcommands
	curOpts := s.Options
	curArgs := s.Args

	// Descend into matched subcommands, leaving `i` at the first token that's
	// a positional arg (or len(rest) if all consumed).
	i := 0
	for i < len(rest) {
		tok := rest[i]
		if strings.HasPrefix(tok, "-") {
			i++
			continue
		}
		var matched *Subcommand
		for j := range curSubs {
			if namesContain(curSubs[j].Name, tok) {
				matched = &curSubs[j]
				break
			}
		}
		if matched == nil {
			break
		}
		curSubs = matched.Subcommands
		curOpts = matched.Options
		curArgs = matched.Args
		i++
	}

	// Number of positional args already typed (i.e. completed tokens past
	// the descent breakpoint that aren't option flags). The trailing `prefix`
	// itself is the NEXT positional, still being typed.
	positionalIdx := 0
	for j := i; j < len(rest); j++ {
		if !strings.HasPrefix(rest[j], "-") {
			positionalIdx++
		}
	}

	// Order: subcommands first, then generator results (files/folders),
	// then option flags. This keeps filesystem entries near the top for
	// commands where path args are the primary use (ls, vim, cat, cd…).
	var out []Suggestion
	for _, sub := range curSubs {
		for _, n := range sub.Name {
			if strings.HasPrefix(n, prefix) {
				out = append(out, Suggestion{
					Value:       n,
					Description: sub.Description,
					Kind:        "subcommand",
				})
			}
		}
	}
	if arg := pickArg(curArgs, positionalIdx); arg != nil && len(arg.Template) > 0 {
		out = append(out, RunTemplates(arg.Template, cwd, prefix)...)
	}
	for _, opt := range curOpts {
		for _, n := range opt.Name {
			if strings.HasPrefix(n, prefix) {
				out = append(out, Suggestion{
					Value:       n,
					Description: opt.Description,
					Kind:        "option",
				})
			}
		}
	}
	return out
}

// pickArg returns the Arg whose positional slot the user is currently typing
// into. Returns nil when there are no args defined or the user has typed past
// a non-variadic tail.
func pickArg(args Args, idx int) *Arg {
	if len(args) == 0 {
		return nil
	}
	if idx < len(args) {
		return &args[idx]
	}
	last := &args[len(args)-1]
	if last.IsVariadic {
		return last
	}
	return nil
}

func namesContain(names Names, tok string) bool {
	for _, n := range names {
		if n == tok {
			return true
		}
	}
	return false
}
