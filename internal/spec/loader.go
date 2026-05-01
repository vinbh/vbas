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
func (l *Loader) Load(cmd string) (*Spec, error) {
	if s, ok := l.cache[cmd]; ok {
		return s, nil
	}
	path := filepath.Join(l.dir, cmd+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			l.cache[cmd] = nil
			return nil, nil
		}
		return nil, err
	}
	var s Spec
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	l.cache[cmd] = &s
	return &s, nil
}
