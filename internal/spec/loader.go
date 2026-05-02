package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Loader reads specs from a directory of JSON files keyed by command name.
type Loader struct {
	dir   string
	cache map[string]*Spec
}

func NewLoader(dir string) *Loader {
	return &Loader{dir: dir, cache: map[string]*Spec{}}
}

// Load returns the spec for cmd, or (nil, nil) if no spec exists.
// Errors are returned only for IO/parse failures, not missing specs.
//
// Lookup order:
//
//  1. <dir>/<cmd>.json     — hand-rolled or hand-tuned spec
//  2. <dir>/fig/<cmd>.json — auto-imported from withfig/autocomplete (M5+)
//
// Hand-rolled wins so users can override an imported spec without forking
// the Fig catalog.
func (l *Loader) Load(cmd string) (*Spec, error) {
	if s, ok := l.cache[cmd]; ok {
		return s, nil
	}
	candidates := []string{
		filepath.Join(l.dir, cmd+".json"),
		filepath.Join(l.dir, "fig", cmd+".json"),
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		var s Spec
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("parse %s: %w", p, err)
		}
		l.cache[cmd] = &s
		return &s, nil
	}
	l.cache[cmd] = nil
	return nil, nil
}
