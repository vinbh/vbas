// Package vbas holds the embedded assets bundled into the vbas binary.
//
// Having this file at the module root is intentional: //go:embed paths are
// relative to the file containing the directive and cannot contain "..".
// Specs live at specs/ and hooks at shell/; both are siblings of this file.
//
// These variables are consumed by "vbas setup", which extracts them to
// ~/.config/vbas/ so that go-install users don't need a separate download.
package vbas

import "embed"

// EmbeddedSpecs contains the full specs/ directory tree (hand-rolled +
// withfig/autocomplete imports). Paths are rooted at "specs/".
//
//go:embed specs
var EmbeddedSpecs embed.FS

// EmbeddedHooks contains the shell integration scripts.
//
//go:embed shell/zsh/vbas.zsh shell/bash/vbas.bash
var EmbeddedHooks embed.FS
