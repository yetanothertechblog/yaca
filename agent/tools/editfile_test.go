package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteEditFile(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("edits file successfully", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test.txt")
		content := "Hello World\nGoodbye World\n"
		os.WriteFile(testFile, []byte(content), 0644)

		args := EditFileArgs{
			FilePath:  testFile,
			OldString: "Hello World",
			NewString: "Hello Go",
		}
		result, err := executeEditFile(args, tempDir)
		if err != nil {
			t.Fatalf("executeEditFile failed: %v", err)
		}

		// Verify file was edited
		modifiedContent, _ := os.ReadFile(testFile)
		modifiedStr := string(modifiedContent)
		if strings.Contains(modifiedStr, "Hello World") {
			t.Error("Old string should be replaced")
		}
		if !strings.Contains(modifiedStr, "Hello Go") {
			t.Error("New string should be present")
		}
		if !strings.Contains(result.Output, testFile) {
			t.Error("Result should contain file path")
		}
	})

	t.Run("returns error for empty file_path", func(t *testing.T) {
		args := EditFileArgs{
			FilePath:  "",
			OldString: "old",
			NewString: "new",
		}
		_, err := executeEditFile(args, tempDir)
		if err == nil {
			t.Error("Expected error for empty file_path")
		}
		toolErr, ok := err.(*ToolError)
		if !ok {
			t.Error("Expected ToolError type")
		}
		if toolErr.Code != ErrMissingField {
			t.Errorf("Expected code %s, got %s", ErrMissingField, toolErr.Code)
		}
	})

	t.Run("returns error for empty old_string", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test2.txt")
		os.WriteFile(testFile, []byte("content"), 0644)

		args := EditFileArgs{
			FilePath:  testFile,
			OldString: "",
			NewString: "new",
		}
		_, err := executeEditFile(args, tempDir)
		if err == nil {
			t.Error("Expected error for empty old_string")
		}
		toolErr, ok := err.(*ToolError)
		if !ok {
			t.Error("Expected ToolError type")
		}
		if toolErr.Code != ErrMissingField {
			t.Errorf("Expected code %s, got %s", ErrMissingField, toolErr.Code)
		}
	})

	t.Run("returns error for identical strings", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test3.txt")
		os.WriteFile(testFile, []byte("content"), 0644)

		args := EditFileArgs{
			FilePath:  testFile,
			OldString: "same",
			NewString: "same",
		}
		_, err := executeEditFile(args, tempDir)
		if err == nil {
			t.Error("Expected error for identical strings")
		}
		toolErr, ok := err.(*ToolError)
		if !ok {
			t.Error("Expected ToolError type")
		}
		if toolErr.Code != ErrIdenticalContent {
			t.Errorf("Expected code %s, got %s", ErrIdenticalContent, toolErr.Code)
		}
	})

	t.Run("returns error for nonexistent file", func(t *testing.T) {
		args := EditFileArgs{
			FilePath:  "nonexistent.txt",
			OldString: "old",
			NewString: "new",
		}
		_, err := executeEditFile(args, tempDir)
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
		toolErr, ok := err.(*ToolError)
		if !ok {
			t.Error("Expected ToolError type")
		}
		if toolErr.Code != ErrFileNotFound {
			t.Errorf("Expected code %s, got %s", ErrFileNotFound, toolErr.Code)
		}
	})

	t.Run("returns error when string not found", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test4.txt")
		os.WriteFile(testFile, []byte("Hello World"), 0644)

		args := EditFileArgs{
			FilePath:  testFile,
			OldString: "Goodbye",
			NewString: "Hello",
		}
		_, err := executeEditFile(args, tempDir)
		if err == nil {
			t.Error("Expected error when string not found")
		}
		toolErr, ok := err.(*ToolError)
		if !ok {
			t.Error("Expected ToolError type")
		}
		if toolErr.Code != ErrStringNotFound {
			t.Errorf("Expected code %s, got %s", ErrStringNotFound, toolErr.Code)
		}
	})

	t.Run("returns error when string not unique", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test5.txt")
		content := "line1\nline1\nline3\n"
		os.WriteFile(testFile, []byte(content), 0644)

		args := EditFileArgs{
			FilePath:  testFile,
			OldString: "line1",
			NewString: "line2",
		}
		_, err := executeEditFile(args, tempDir)
		if err == nil {
			t.Error("Expected error when string not unique")
		}
		toolErr, ok := err.(*ToolError)
		if !ok {
			t.Error("Expected ToolError type")
		}
		if toolErr.Code != ErrStringNotUnique {
			t.Errorf("Expected code %s, got %s", ErrStringNotUnique, toolErr.Code)
		}
	})

	t.Run("edits multiline strings", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test6.txt")
		content := "line1\nline2\nline3\n"
		os.WriteFile(testFile, []byte(content), 0644)

		args := EditFileArgs{
			FilePath:  testFile,
			OldString: "line1\nline2",
			NewString: "lineA\nlineB",
		}
		result, err := executeEditFile(args, tempDir)
		if err != nil {
			t.Fatalf("executeEditFile failed: %v", err)
		}

		modifiedContent, _ := os.ReadFile(testFile)
		modifiedStr := string(modifiedContent)
		if !strings.Contains(modifiedStr, "lineA\nlineB") {
			t.Error("Multiline replacement should work")
		}
		if !strings.Contains(result.Output, "new_string") {
			t.Error("Result should be JSON formatted with 'new_string' field")
		}
	})
}
