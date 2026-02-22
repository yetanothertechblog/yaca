package lsp

import (
	"encoding/json"
	"testing"
)

func TestDiagnostic_JSONFieldNames(t *testing.T) {
	d := Diagnostic{
		Range:    Range{Start: Position{Line: 1, Character: 5}, End: Position{Line: 1, Character: 10}},
		Severity: SeverityError,
		Message:  "undefined variable",
		Source:   "gopls",
	}
	data, _ := json.Marshal(d)

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	// Verify wire-format field names match LSP spec
	if _, ok := raw["range"]; !ok {
		t.Error("missing 'range' field")
	}
	if _, ok := raw["severity"]; !ok {
		t.Error("missing 'severity' field")
	}
	if raw["severity"] != float64(SeverityError) {
		t.Errorf("severity = %v, want %d", raw["severity"], SeverityError)
	}
}

func TestDiagnostic_OmitsEmptySource(t *testing.T) {
	d := Diagnostic{Severity: SeverityWarning, Message: "warning"}
	data, _ := json.Marshal(d)

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	if _, ok := raw["source"]; ok {
		t.Error("empty source should be omitted")
	}
}

func TestPublishDiagnosticsParams_RoundTrip(t *testing.T) {
	params := PublishDiagnosticsParams{
		URI: "file:///tmp/test.go",
		Diagnostics: []Diagnostic{
			{Severity: SeverityError, Message: "err1"},
			{Severity: SeverityWarning, Message: "warn1"},
		},
	}
	data, _ := json.Marshal(params)

	var decoded PublishDiagnosticsParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.URI != "file:///tmp/test.go" {
		t.Errorf("URI = %q", decoded.URI)
	}
	if len(decoded.Diagnostics) != 2 {
		t.Errorf("Diagnostics count = %d, want 2", len(decoded.Diagnostics))
	}
	if decoded.Diagnostics[0].Message != "err1" {
		t.Errorf("first diagnostic message = %q", decoded.Diagnostics[0].Message)
	}
}
