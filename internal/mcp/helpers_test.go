package mcp

import (
	"encoding/json"
	"testing"
)

func TestOptionalRaw_Object(t *testing.T) {
	in := map[string]any{
		"description": map[string]any{
			"type":    "doc",
			"version": 1,
			"content": []any{},
		},
	}
	raw, present, err := optionalRaw(in, "description")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !present {
		t.Fatal("expected present=true")
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("expected JSON object, got %q: %v", raw, err)
	}
	if got["type"] != "doc" || got["version"] != float64(1) {
		t.Errorf("expected ADF doc, got %v", got)
	}
}

func TestOptionalRaw_String(t *testing.T) {
	raw, present, err := optionalRaw(map[string]any{"description": "hi"}, "description")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !present {
		t.Fatal("expected present=true")
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("expected JSON string, got %q: %v", raw, err)
	}
	if s != "hi" {
		t.Errorf("expected hi, got %q", s)
	}
}

func TestOptionalRaw_Missing(t *testing.T) {
	raw, present, err := optionalRaw(map[string]any{}, "description")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if present {
		t.Error("expected present=false when missing")
	}
	if raw != nil {
		t.Errorf("expected nil raw when missing, got %q", raw)
	}
}

func TestOptionalRaw_WrongType(t *testing.T) {
	_, _, err := optionalRaw(map[string]any{"description": 42}, "description")
	if err == nil {
		t.Fatal("expected error when description is a number")
	}
}
