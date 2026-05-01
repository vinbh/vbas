// Package spec defines the completion specification format used by vbas.
//
// The schema is intentionally a strict subset of Fig's autocomplete spec
// (https://fig.io/docs/reference/spec) so transpiled Fig specs can drop in
// later without schema migration. Dynamic generators and option arg parsing
// are deferred to later milestones.
package spec

import (
	"bytes"
	"encoding/json"
)

// Spec is a top-level completion specification for a single command.
type Spec struct {
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Subcommands []Subcommand `json:"subcommands,omitempty"`
	Options     []Option     `json:"options,omitempty"`
	Args        []Arg        `json:"args,omitempty"`
}

// Subcommand is a nested command under a parent (e.g., "checkout" under "git").
type Subcommand struct {
	Name        Names        `json:"name"`
	Description string       `json:"description,omitempty"`
	Subcommands []Subcommand `json:"subcommands,omitempty"`
	Options     []Option     `json:"options,omitempty"`
	Args        []Arg        `json:"args,omitempty"`
}

// Option is a flag like "-m" or "--message".
type Option struct {
	Name        Names  `json:"name"`
	Description string `json:"description,omitempty"`
	Args        []Arg  `json:"args,omitempty"`
}

// Arg describes a positional argument.
type Arg struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	IsOptional  bool   `json:"isOptional,omitempty"`
}

// Names is either a single string or array of strings, matching Fig's
// pattern where short and long forms share an Option entry.
type Names []string

func (n *Names) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '"' {
		var single string
		if err := json.Unmarshal(data, &single); err != nil {
			return err
		}
		*n = Names{single}
		return nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	*n = list
	return nil
}
