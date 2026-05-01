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
func Match(s *Spec, buffer string) []Suggestion {
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

	for _, tok := range rest {
		if strings.HasPrefix(tok, "-") {
			continue
		}
		var matched *Subcommand
		for i := range curSubs {
			if namesContain(curSubs[i].Name, tok) {
				matched = &curSubs[i]
				break
			}
		}
		if matched == nil {
			break
		}
		curSubs = matched.Subcommands
		curOpts = matched.Options
	}

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

func namesContain(names Names, tok string) bool {
	for _, n := range names {
		if n == tok {
			return true
		}
	}
	return false
}
