package tools

import (
	"errors"
	"testing"
)

func TestToolError_Error(t *testing.T) {
	err := &ToolError{Code: "TEST_ERROR", Message: "something broke"}
	if err.Error() != "TEST_ERROR: something broke" {
		t.Errorf("Error() = %q", err.Error())
	}
}

func TestToolError_DetailsExcludedFromError(t *testing.T) {
	err := NewToolErrorWithDetails("CODE", "msg", "secret details")
	if err.Error() != "CODE: msg" {
		t.Errorf("Error() should not include details, got %q", err.Error())
	}
	if err.Details != "secret details" {
		t.Errorf("Details field lost, got %q", err.Details)
	}
}

func TestToolError_ImplementsErrorInterface(t *testing.T) {
	var err error = NewToolError("TEST", "msg")
	var toolErr *ToolError
	if !errors.As(err, &toolErr) {
		t.Error("ToolError should be unwrappable via errors.As")
	}
}
