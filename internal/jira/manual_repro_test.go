package jira

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestManualRepro_ADFFromCallerBytes is a manual end-to-end demonstration of
// the polymorphic Description contract. It writes a small Go program inline
// (instead of a real Jira round-trip) and prints the wire output so a human
// can verify the ADF doc object is byte-equal to the caller's input.
//
// This satisfies task 5.4 of the sdd-tasks artifact. The actual Jira call
// happens in sdd-verify, not here.
func TestManualRepro_ADFFromCallerBytes(t *testing.T) {
	// Simulate the caller-supplied ADF that would arrive at the handler.
	caller := []byte(`{"type":"doc","version":1,"content":[{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"MC-658 Repro"}]},{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Headings render"}]}]}]},{"type":"paragraph","content":[{"type":"text","text":"Inline body."}]}]}`)

	req := UpdateIssueRequest{
		Summary:     "MC-658 — fix ADF round-trip",
		Description: json.RawMessage(caller),
	}
	wire, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	t.Logf("wire output:\n%s", wire)

	// Extract the description payload from the wire output and check it is
	// byte-equal to the caller's input.
	var envelope struct {
		Fields struct {
			Description json.RawMessage `json:"description"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(wire, &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if string(envelope.Fields.Description) != string(caller) {
		t.Errorf("description NOT byte-equal:\n  want: %s\n  got:  %s", caller, envelope.Fields.Description)
	}
	fmt.Printf("\n=== MANUAL REPRO ===\nCaller ADF: %d bytes\nWire output: %d bytes\nByte-equal: %v\nDescription: %s\n",
		len(caller), len(wire), string(caller) == string(envelope.Fields.Description), envelope.Fields.Description)
}

func TestManualRepro_StringStillWrapsAsADF(t *testing.T) {
	req := UpdateIssueRequest{
		Summary:     "Legacy",
		Description: json.RawMessage(`"hello world"`),
	}
	wire, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var envelope struct {
		Fields struct {
			Description struct {
				Type    string `json:"type"`
				Version int    `json:"version"`
				Content []struct {
					Type    string `json:"type"`
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
				} `json:"content"`
			} `json:"description"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(wire, &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if envelope.Fields.Description.Type != "doc" || envelope.Fields.Description.Version != 1 {
		t.Errorf("expected ADF doc shape, got %+v", envelope.Fields.Description)
	}
	if len(envelope.Fields.Description.Content) != 1 {
		t.Errorf("expected 1 paragraph, got %d", len(envelope.Fields.Description.Content))
	}
	fmt.Printf("\n=== LEGACY STRING REPRO ===\nWire output: %s\n", wire)
}
