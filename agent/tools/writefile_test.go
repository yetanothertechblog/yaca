package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteWriteFile(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("creates new file successfully", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "newfile.txt")
		content := "Hello, World!"

		args := WriteFileArgs{
			FilePath: testFile,
			Content:  content,
		}
		result, err := executeWriteFile(args, tempDir)
		if err != nil {
			t.Fatalf("executeWriteFile failed: %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Error("File should have been created")
		}

		// Verify content
		data, _ := os.ReadFile(testFile)
		if string(data) != content {
			t.Errorf("Expected content '%s', got '%s'", content, string(data))
		}

		// Verify result JSON
		var writeResult WriteFileResult
		if err := json.Unmarshal([]byte(result.Output), &writeResult); err != nil {
			t.Fatalf("Failed to parse result JSON: %v", err)
		}
		if !writeResult.IsNewFile {
			t.Error("IsNewFile should be true for new files")
		}
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "existing.txt")
		oldContent := "old content"
		newContent := "new content"

		os.WriteFile(testFile, []byte(oldContent), 0644)

		args := WriteFileArgs{
			FilePath: testFile,
			Content:  newContent,
		}
		result, err := executeWriteFile(args, tempDir)
		if err != nil {
			t.Fatalf("executeWriteFile failed: %v", err)
		}

		// Verify file was overwritten
		data, _ := os.ReadFile(testFile)
		if string(data) != newContent {
			t.Errorf("Expected new content '%s', got '%s'", newContent, string(data))
		}

		// Verify result JSON
		var writeResult WriteFileResult
		if err := json.Unmarshal([]byte(result.Output), &writeResult); err != nil {
			t.Fatalf("Failed to parse result JSON: %v", err)
		}
		if writeResult.IsNewFile {
			t.Error("IsNewFile should be false for existing files")
		}
	})

	t.Run("returns error for empty file_path", func(t *testing.T) {
		args := WriteFileArgs{
			FilePath: "",
			Content:  "content",
		}
		_, err := executeWriteFile(args, tempDir)
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

	t.Run("creates parent directories", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "subdir", "nested", "file.txt")

		args := WriteFileArgs{
			FilePath: testFile,
			Content:  "nested content",
		}
		_, err := executeWriteFile(args, tempDir)
		if err != nil {
			t.Fatalf("executeWriteFile failed: %v", err)
		}

		// Verify parent directories were created
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Error("File should have been created in nested directory")
		}

		data, _ := os.ReadFile(testFile)
		if string(data) != "nested content" {
			t.Errorf("Expected 'nested content', got '%s'", string(data))
		}
	})

	t.Run("handles relative paths", func(t *testing.T) {
		testFile := "relative.txt"
		content := "relative content"

		args := WriteFileArgs{
			FilePath: testFile,
			Content:  content,
		}
		_, err := executeWriteFile(args, tempDir)
		if err != nil {
			t.Fatalf("executeWriteFile failed: %v", err)
		}

		// Verify file was created at relative path
		data, _ := os.ReadFile(filepath.Join(tempDir, testFile))
		if string(data) != content {
			t.Errorf("Expected '%s', got '%s'", content, string(data))
		}
	})

	t.Run("handles empty content", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "empty.txt")

		args := WriteFileArgs{
			FilePath: testFile,
			Content:  "",
		}
		_, err := executeWriteFile(args, tempDir)
		if err != nil {
			t.Fatalf("executeWriteFile failed: %v", err)
		}

		data, _ := os.ReadFile(testFile)
		if string(data) != "" {
			t.Errorf("Expected empty content, got '%s'", string(data))
		}
	})

	t.Run("result contains correct JSON fields", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "json_test.txt")
		content := "test content"

		args := WriteFileArgs{
			FilePath: testFile,
			Content:  content,
		}
		result, err := executeWriteFile(args, tempDir)
		if err != nil {
			t.Fatalf("executeWriteFile failed: %v", err)
		}

		var writeResult WriteFileResult
		if err := json.Unmarshal([]byte(result.Output), &writeResult); err != nil {
			t.Fatalf("Failed to parse result JSON: %v", err)
		}

		if writeResult.FilePath != testFile {
			t.Errorf("Expected FilePath '%s', got '%s'", testFile, writeResult.FilePath)
		}
		if writeResult.NewContent != content {
			t.Errorf("Expected NewContent '%s', got '%s'", content, writeResult.NewContent)
		}
		if !strings.Contains(result.Output, "is_new_file") {
			t.Error("Result JSON should contain is_new_file field")
		}
	})
}
