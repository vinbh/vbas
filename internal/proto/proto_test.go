package proto

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/vinbh/peek/internal/spec"
)

func TestRequestRoundTrip(t *testing.T) {
	in := Request{Op: "complete", Buffer: "git c", Cursor: 5}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out Request
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Errorf("round-trip mismatch: in=%+v out=%+v", in, out)
	}
}

func TestResponseRoundTrip(t *testing.T) {
	in := Response{Suggestions: []spec.Suggestion{
		{Value: "checkout", Description: "Switch branches", Kind: "subcommand"},
		{Value: "commit", Description: "Record changes", Kind: "subcommand"},
	}}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(in); err != nil {
		t.Fatal(err)
	}
	var out Response
	if err := json.NewDecoder(&buf).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Suggestions) != 2 || out.Suggestions[0].Value != "checkout" {
		t.Errorf("unexpected decoded response: %+v", out)
	}
}

func TestErrorResponse(t *testing.T) {
	in := Response{Error: "no such spec"}
	data, _ := json.Marshal(in)
	if !bytes.Contains(data, []byte(`"error":"no such spec"`)) {
		t.Errorf("error field not in JSON: %s", data)
	}
	if bytes.Contains(data, []byte(`"suggestions"`)) {
		t.Errorf("suggestions should be omitted when nil: %s", data)
	}
}
