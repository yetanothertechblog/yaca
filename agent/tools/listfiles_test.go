package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteListFiles(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("lists files in current directory", func(t *testing.T) {
		// Create test files
		os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("content1"), 0644)
		os.WriteFile(filepath.Join(tempDir, "file2.go"), []byte("content2"), 0644)
		os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)

		args := ListFilesArgs{Path: tempDir}
		result, err := executeListFiles(args, tempDir)
		if err != nil {
			t.Fatalf("executeListFiles failed: %v", err)
		}

		output := result.Output
		if !strings.Contains(output, "file1.txt") {
			t.Error("Output should contain file1.txt")
		}
		if !strings.Contains(output, "file2.go") {
			t.Error("Output should contain file2.go")
		}
		if !strings.Contains(output, "subdir/") {
			t.Error("Output should contain subdir/")
		}
	})

	t.Run("uses working directory when path is empty", func(t *testing.T) {
		os.WriteFile(filepath.Join(tempDir, "working.txt"), []byte("test"), 0644)

		args := ListFilesArgs{Path: ""}
		result, err := executeListFiles(args, tempDir)
		if err != nil {
			t.Fatalf("executeListFiles failed: %v", err)
		}

		if !strings.Contains(result.Output, "working.txt") {
			t.Error("Output should contain working.txt")
		}
	})

	t.Run("lists files recursively", func(t *testing.T) {
		// Create nested structure
		os.MkdirAll(filepath.Join(tempDir, "subdir1", "subdir2"), 0755)
		os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("test1"), 0644)
		os.WriteFile(filepath.Join(tempDir, "subdir1", "file2.txt"), []byte("test2"), 0644)
		os.WriteFile(filepath.Join(tempDir, "subdir1", "subdir2", "file3.txt"), []byte("test3"), 0644)

		args := ListFilesArgs{Path: tempDir, Recursive: true}
		result, err := executeListFiles(args, tempDir)
		if err != nil {
			t.Fatalf("executeListFiles failed: %v", err)
		}

		output := result.Output
		if !strings.Contains(output, "file1.txt") {
			t.Error("Output should contain file1.txt")
		}
		if !strings.Contains(output, "subdir1/file2.txt") {
			t.Error("Output should contain subdir1/file2.txt")
		}
		if !strings.Contains(output, "subdir1/subdir2/file3.txt") {
			t.Error("Output should contain nested file3.txt")
		}
	})

	t.Run("returns error for nonexistent path", func(t *testing.T) {
		args := ListFilesArgs{Path: "/nonexistent/path"}
		_, err := executeListFiles(args, tempDir)
		if err == nil {
			t.Error("Expected error for nonexistent path")
		}
		toolErr, ok := err.(*ToolError)
		if !ok {
			t.Error("Expected ToolError type")
		}
		if toolErr.Code != ErrFileNotFound {
			t.Errorf("Expected code %s, got %s", ErrFileNotFound, toolErr.Code)
		}
	})

	t.Run("returns error when path is not a directory", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "notadir.txt")
		os.WriteFile(testFile, []byte("content"), 0644)

		args := ListFilesArgs{Path: testFile}
		_, err := executeListFiles(args, tempDir)
		if err == nil {
			t.Error("Expected error when path is not a directory")
		}
		toolErr, ok := err.(*ToolError)
		if !ok {
			t.Error("Expected ToolError type")
		}
		if toolErr.Code != ErrInvalidArguments {
			t.Errorf("Expected code %s, got %s", ErrInvalidArguments, toolErr.Code)
		}
	})

	t.Run("skips .git directory", func(t *testing.T) {
		gitDir := filepath.Join(tempDir, ".git")
		os.Mkdir(gitDir, 0755)
		os.WriteFile(filepath.Join(gitDir, "config"), []byte("git config"), 0644)

		args := ListFilesArgs{Path: tempDir}
		result, err := executeListFiles(args, tempDir)
		if err != nil {
			t.Fatalf("executeListFiles failed: %v", err)
		}

		if strings.Contains(result.Output, ".git/") {
			t.Error(".git directory should be skipped")
		}
	})

	t.Run("skips .git directory recursively", func(t *testing.T) {
		gitDir := filepath.Join(tempDir, ".git")
		os.MkdirAll(filepath.Join(gitDir, "objects", "pack"), 0755)
		os.WriteFile(filepath.Join(gitDir, "config"), []byte("config"), 0644)

		args := ListFilesArgs{Path: tempDir, Recursive: true}
		result, err := executeListFiles(args, tempDir)
		if err != nil {
			t.Fatalf("executeListFiles failed: %v", err)
		}

		if strings.Contains(result.Output, ".git") {
			t.Error(".git directory and contents should be skipped recursively")
		}
	})

	t.Run("handles relative paths", func(t *testing.T) {
		subdir := "relativesub"
		os.Mkdir(filepath.Join(tempDir, subdir), 0755)
		os.WriteFile(filepath.Join(tempDir, subdir, "file.txt"), []byte("test"), 0644)

		args := ListFilesArgs{Path: subdir}
		result, err := executeListFiles(args, tempDir)
		if err != nil {
			t.Fatalf("executeListFiles failed: %v", err)
		}

		if !strings.Contains(result.Output, "file.txt") {
			t.Error("Should handle relative paths")
		}
	})

	t.Run("handles empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		os.Mkdir(emptyDir, 0755)

		args := ListFilesArgs{Path: emptyDir}
		result, err := executeListFiles(args, tempDir)
		if err != nil {
			t.Fatalf("executeListFiles failed: %v", err)
		}

		// Empty directory produces empty string (no files to list)
		// This is acceptable behavior
		_ = result.Output // Use the result
	})
}
