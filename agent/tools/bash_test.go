package tools

import (
	"strings"
	"testing"
)

func TestExecuteBash(t *testing.T) {
	t.Run("executes echo command successfully", func(t *testing.T) {
		args := BashArgs{Command: "echo 'Hello, World!'"}
		result, err := executeBash(args, "/tmp")
		if err != nil {
			t.Fatalf("executeBash failed: %v", err)
		}

		if !strings.Contains(result.Output, "Hello, World!") {
			t.Errorf("Expected output to contain 'Hello, World!', got '%s'", result.Output)
		}
	})

	t.Run("returns error for empty command", func(t *testing.T) {
		args := BashArgs{Command: ""}
		_, err := executeBash(args, "/tmp")
		if err == nil {
			t.Error("Expected error for empty command")
		}
		toolErr, ok := err.(*ToolError)
		if !ok {
			t.Error("Expected ToolError type")
		}
		if toolErr.Code != ErrMissingField {
			t.Errorf("Expected code %s, got %s", ErrMissingField, toolErr.Code)
		}
	})

	t.Run("executes ls command", func(t *testing.T) {
		args := BashArgs{Command: "ls -la /tmp"}
		result, err := executeBash(args, "/tmp")
		if err != nil {
			t.Fatalf("executeBash failed: %v", err)
		}

		// ls should always produce some output
		if result.Output == "" {
			t.Error("ls command should produce output")
		}
	})

	t.Run("handles command with non-zero exit code", func(t *testing.T) {
		args := BashArgs{Command: "ls /nonexistent_directory_12345"}
		result, err := executeBash(args, "/tmp")
		if err != nil {
			t.Fatalf("executeBash should not return error for non-zero exit code: %v", err)
		}

		// Result should contain error information
		if !strings.Contains(result.Output, "exit status") {
			t.Errorf("Expected output to contain exit status, got '%s'", result.Output)
		}
	})

	t.Run("captures stderr", func(t *testing.T) {
		args := BashArgs{Command: "echo 'error message' >&2"}
		result, err := executeBash(args, "/tmp")
		if err != nil {
			t.Fatalf("executeBash failed: %v", err)
		}

		if !strings.Contains(result.Output, "error message") {
			t.Errorf("Expected output to contain stderr, got '%s'", result.Output)
		}
	})

	t.Run("handles command with no output", func(t *testing.T) {
		args := BashArgs{Command: "true"}
		result, err := executeBash(args, "/tmp")
		if err != nil {
			t.Fatalf("executeBash failed: %v", err)
		}

		// Commands with no output should return "(no output)"
		if result.Output != "(no output)" {
			t.Errorf("Expected '(no output)', got '%s'", result.Output)
		}
	})

	t.Run("respects working directory", func(t *testing.T) {
		// Test with a different working directory
		args := BashArgs{Command: "pwd"}
		result, err := executeBash(args, "/")
		if err != nil {
			t.Fatalf("executeBash failed: %v", err)
		}

		if !strings.Contains(result.Output, "/") {
			t.Errorf("Expected pwd to show '/', got '%s'", result.Output)
		}
	})

	t.Run("handles multi-line output", func(t *testing.T) {
		args := BashArgs{Command: "printf 'line1\\nline2\\nline3'"}
		result, err := executeBash(args, "/tmp")
		if err != nil {
			t.Fatalf("executeBash failed: %v", err)
		}

		if !strings.Contains(result.Output, "line1") ||
			!strings.Contains(result.Output, "line2") ||
			!strings.Contains(result.Output, "line3") {
			t.Errorf("Expected all three lines in output, got '%s'", result.Output)
		}
	})
}
