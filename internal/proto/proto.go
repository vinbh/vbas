// Package proto defines the wire format between the vbas client and the
// long-running vbas daemon. Communication is newline-delimited JSON over
// a Unix domain socket: one Request per connection followed by one
// Response, then the daemon closes.
//
// Keeping it line-delimited (not length-prefixed binary) means you can
// debug a stuck daemon with plain `nc -U`.
package proto

import "github.com/vinbh/vbas/internal/spec"

// Request is what the client sends to the daemon.
type Request struct {
	Op     string `json:"op"`               // "complete" today; "ping"/"shutdown"/etc. later
	Buffer string `json:"buffer,omitempty"` // current command line for op="complete"
	Cursor int    `json:"cursor,omitempty"` // reserved
}

// Response is what the daemon writes back. Exactly one of Suggestions
// or Error is populated.
type Response struct {
	Suggestions []spec.Suggestion `json:"suggestions,omitempty"`
	Error       string            `json:"error,omitempty"`
}
