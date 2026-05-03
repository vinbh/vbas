package spec

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunTemplates expands the built-in Fig generators "filepaths" / "folders"
// against the client's working directory, filtered by the user's partial
// token. Other templates ("history", and any unrecognized value) return nil
// — those land in M6.
//
// token is whatever the user has typed for the current arg position (empty
// when buffer ends in space). cwd is the client-supplied working directory;
// when empty, relative tokens fall back to the daemon's process cwd.
//
// Returned suggestion values are constructed to replace the trailing token
// verbatim — so if the user typed "~/Doc", suggestions look like "~/Documents/"
// (the tilde is preserved, not expanded). Directories always end in "/" so
// the cascade keeps drilling.
func RunTemplates(templates Templates, cwd, token string) []Suggestion {
	wantFiles, wantDirs := false, false
	for _, t := range templates {
		switch t {
		case "filepaths":
			wantFiles, wantDirs = true, true
		case "folders":
			wantDirs = true
		}
	}
	if !wantFiles && !wantDirs {
		return nil
	}

	listDir, displayPrefix, nameFilter := splitToken(token, cwd)
	if listDir == "" {
		return nil
	}

	entries, err := os.ReadDir(listDir)
	if err != nil {
		return nil
	}

	// Hidden files are skipped unless the user is explicitly typing a "."
	// prefix — otherwise `cat <Tab>` would dump every dotfile in $HOME.
	showHidden := strings.HasPrefix(nameFilter, ".")

	out := make([]Suggestion, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		if nameFilter != "" && !strings.HasPrefix(name, nameFilter) {
			continue
		}
		isDir := e.IsDir()
		// Resolve symlinks once so a symlinked dir behaves like a dir.
		if !isDir && e.Type()&os.ModeSymlink != 0 {
			if info, err := os.Stat(filepath.Join(listDir, name)); err == nil && info.IsDir() {
				isDir = true
			}
		}
		if isDir {
			if !wantDirs {
				continue
			}
			out = append(out, Suggestion{
				Value: displayPrefix + name + "/",
				Kind:  "arg",
			})
		} else {
			if !wantFiles {
				continue
			}
			out = append(out, Suggestion{
				Value: displayPrefix + name,
				Kind:  "arg",
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Value < out[j].Value })
	return out
}

// splitToken decomposes a partial path token into:
//   - listDir: actual filesystem directory to read
//   - displayPrefix: what to prepend to each entry name in the suggestion
//     value, preserving the user's typed form (so "~/Doc" stays a "~"-prefixed
//     suggestion rather than an expanded "/home/.../Documents")
//   - nameFilter: the basename prefix to filter entries by
//
// Empty cwd falls back to the daemon process cwd.
func splitToken(token, cwd string) (listDir, displayPrefix, nameFilter string) {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	// "" → list cwd, no filter, no display prefix.
	if token == "" {
		return cwd, "", ""
	}

	// Tilde expansion happens for filesystem reads only — the displayed
	// suggestion preserves the tilde so the buffer stays compact.
	expand := func(p string) string {
		if p == "~" {
			if home, err := os.UserHomeDir(); err == nil {
				return home
			}
			return p
		}
		if strings.HasPrefix(p, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				return filepath.Join(home, p[2:])
			}
		}
		return p
	}

	// Treat bare "~" like "~/" — list $HOME.
	if token == "~" {
		return expand(token), "~/", ""
	}

	// Trailing slash → token IS the directory; nothing to filter.
	if strings.HasSuffix(token, "/") {
		listDir = expand(token)
		if !filepath.IsAbs(listDir) {
			listDir = filepath.Join(cwd, listDir)
		}
		return listDir, token, ""
	}

	// Otherwise token has a final basename. Split on the original token so
	// the visible prefix preserves what the user typed; resolve listDir
	// from the expanded form.
	origDir, base := filepath.Split(token)
	expandedDir, _ := filepath.Split(expand(token))

	if origDir == "" {
		// Just a bare name like "foo" — listing happens in cwd.
		return cwd, "", base
	}

	listDir = expandedDir
	if !filepath.IsAbs(listDir) {
		listDir = filepath.Join(cwd, listDir)
	}
	return listDir, origDir, base
}
