package restic

import "testing"

type jsonTestStruct struct {
	Value int `json:"value"`
}

func TestParseJSONSkippingWarnings_NoWarnings(t *testing.T) {
	var result jsonTestStruct
	parsed, skipped, err := parseJSONSkippingWarnings([]byte("{\"value\":1}"), &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skipped {
		t.Fatalf("expected no skipped warnings")
	}
	if string(parsed) != "{\"value\":1}" {
		t.Fatalf("unexpected parsed output: %q", string(parsed))
	}
	if result.Value != 1 {
		t.Fatalf("unexpected parsed value: %d", result.Value)
	}
}

func TestParseJSONSkippingWarnings_WithWarnings(t *testing.T) {
	var result jsonTestStruct
	input := []byte("warning: foo\n{\"value\":2}\n")
	parsed, skipped, err := parseJSONSkippingWarnings(input, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !skipped {
		t.Fatalf("expected skipped warnings")
	}
	if string(parsed) != "{\"value\":2}" {
		t.Fatalf("unexpected parsed output: %q", string(parsed))
	}
	if result.Value != 2 {
		t.Fatalf("unexpected parsed value: %d", result.Value)
	}
}

func TestParseJSONSkippingWarnings_NoNewline(t *testing.T) {
	var result jsonTestStruct
	input := []byte("warning without newline {\"value\":3}")
	if _, _, err := parseJSONSkippingWarnings(input, &result); err == nil {
		t.Fatalf("expected error when JSON cannot be located")
	}
}
