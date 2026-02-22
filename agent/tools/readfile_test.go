package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteReadFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	t.Run("reads file successfully", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test.txt")
		content := "line1\nline2\nline3\n"
		os.WriteFile(testFile, []byte(content), 0644)

		args := ReadFileArgs{FilePath: testFile}
		result, err := executeReadFile(args, tempDir)
		if err != nil {
			t.Fatalf("executeReadFile failed: %v", err)
		}

		// Note: strings.Split adds an extra empty line for trailing newlines
		if !strings.Contains(result.Output, "line1") ||
			!strings.Contains(result.Output, "line2") ||
			!strings.Contains(result.Output, "line3") {
			t.Errorf("Output should contain all three lines, got:\n%s", result.Output)
		}
	})

	t.Run("reads file with offset", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "offset_test.txt")
		content := "line1\nline2\nline3\nline4\nline5\n"
		os.WriteFile(testFile, []byte(content), 0644)

		args := ReadFileArgs{FilePath: testFile, Offset: 2}
		result, err := executeReadFile(args, tempDir)
		if err != nil {
			t.Fatalf("executeReadFile failed: %v", err)
		}

		if result.Output == "" {
			t.Error("Expected non-empty output")
		}
		// Should show lines 2-5
		if !strings.Contains(result.Output, "line2") || !strings.Contains(result.Output, "line5") {
			t.Errorf("Output should contain lines 2-5, got:\n%s", result.Output)
		}
	})

	t.Run("reads file with limit", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "limit_test.txt")
		content := "line1\nline2\nline3\nline4\nline5\n"
		os.WriteFile(testFile, []byte(content), 0644)

		args := ReadFileArgs{FilePath: testFile, Limit: 2}
		result, err := executeReadFile(args, tempDir)
		if err != nil {
			t.Fatalf("executeReadFile failed: %v", err)
		}

		// Should only show 2 lines
		if !strings.Contains(result.Output, "line1") || !strings.Contains(result.Output, "line2") {
			t.Errorf("Output should contain lines 1-2, got:\n%s", result.Output)
		}
		if strings.Contains(result.Output, "line3") {
			t.Error("Output should not contain line3")
		}
	})

	t.Run("reads file with offset and limit", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "both_test.txt")
		content := "line1\nline2\nline3\nline4\nline5\n"
		os.WriteFile(testFile, []byte(content), 0644)

		args := ReadFileArgs{FilePath: testFile, Offset: 2, Limit: 2}
		result, err := executeReadFile(args, tempDir)
		if err != nil {
			t.Fatalf("executeReadFile failed: %v", err)
		}

		// Should show lines 2-3
		if !strings.Contains(result.Output, "line2") || !strings.Contains(result.Output, "line3") {
			t.Errorf("Output should contain lines 2-3, got:\n%s", result.Output)
		}
		if strings.Contains(result.Output, "line1") || strings.Contains(result.Output, "line4") {
			t.Error("Output should not contain line1 or line4")
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		args := ReadFileArgs{FilePath: "nonexistent.txt"}
		_, err := executeReadFile(args, tempDir)
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

	t.Run("returns error for empty file_path", func(t *testing.T) {
		args := ReadFileArgs{FilePath: ""}
		_, err := executeReadFile(args, tempDir)
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

	t.Run("handles offset out of bounds", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "short.txt")
		content := "line1\nline2\n"
		os.WriteFile(testFile, []byte(content), 0644)

		args := ReadFileArgs{FilePath: testFile, Offset: 10}
		_, err := executeReadFile(args, tempDir)
		if err == nil {
			t.Error("Expected error for offset out of bounds")
		}
		toolErr, ok := err.(*ToolError)
		if !ok {
			t.Error("Expected ToolError type")
		}
		if toolErr.Code != ErrInvalidArguments {
			t.Errorf("Expected code %s, got %s", ErrInvalidArguments, toolErr.Code)
		}
	})

	t.Run("normalizes negative offset to 1", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "negoffset.txt")
		content := "line1\nline2\n"
		os.WriteFile(testFile, []byte(content), 0644)

		args := ReadFileArgs{FilePath: testFile, Offset: -1}
		result, err := executeReadFile(args, tempDir)
		if err != nil {
			t.Fatalf("executeReadFile failed: %v", err)
		}
		if !strings.Contains(result.Output, "line1") {
			t.Error("Should show first line with negative offset")
		}
	})

	t.Run("handles empty file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "empty.txt")
		os.WriteFile(testFile, []byte(""), 0644)

		args := ReadFileArgs{FilePath: testFile}
		result, err := executeReadFile(args, tempDir)
		if err != nil {
			t.Fatalf("executeReadFile failed: %v", err)
		}
		// Empty file when split by \n produces 1 empty element
		if !strings.Contains(result.Output, "1 total lines") {
			t.Errorf("Should show 1 total line for empty file, got:\n%s", result.Output)
		}
	})

	t.Run("resolves relative paths", func(t *testing.T) {
		testFile := "relative.txt"
		content := "relative content"
		os.WriteFile(filepath.Join(tempDir, testFile), []byte(content), 0644)

		args := ReadFileArgs{FilePath: testFile}
		result, err := executeReadFile(args, tempDir)
		if err != nil {
			t.Fatalf("executeReadFile failed: %v", err)
		}
		if !strings.Contains(result.Output, "relative content") {
			t.Error("Should read file with relative path")
		}
	})
}
