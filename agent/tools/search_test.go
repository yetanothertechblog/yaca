package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteSearch(t *testing.T) {
	t.Run("empty pattern returns error", func(t *testing.T) {
		_, err := executeSearch(SearchArgs{Pattern: ""}, t.TempDir())
		toolErr, ok := err.(*ToolError)
		if !ok || toolErr.Code != ErrMissingField {
			t.Errorf("expected ToolError with code %s, got %v", ErrMissingField, err)
		}
	})

	t.Run("nonexistent path returns error", func(t *testing.T) {
		_, err := executeSearch(SearchArgs{Pattern: "x", Path: "/nonexistent/path/12345"}, t.TempDir())
		toolErr, ok := err.(*ToolError)
		if !ok || toolErr.Code != ErrFileNotFound {
			t.Errorf("expected ToolError with code %s, got %v", ErrFileNotFound, err)
		}
	})

	t.Run("no matches returns message", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "test.go"), []byte("package main"), 0644)

		result, err := executeSearch(SearchArgs{Pattern: "zzz_nonexistent_zzz"}, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Output != "No matches found." {
			t.Errorf("expected 'No matches found.', got %q", result.Output)
		}
	})

	t.Run("finds pattern with file and line number", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "test.go"), []byte("line1\ntarget_xyz\nline3\n"), 0644)

		result, err := executeSearch(SearchArgs{Pattern: "target_xyz"}, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result.Output, "test.go") {
			t.Errorf("output should contain filename, got %q", result.Output)
		}
		if !strings.Contains(result.Output, "target_xyz") {
			t.Errorf("output should contain matched text, got %q", result.Output)
		}
	})

	t.Run("searches txt files too", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "only.txt"), []byte("unique_txt_needle"), 0644)

		result, err := executeSearch(SearchArgs{Pattern: "unique_txt_needle"}, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result.Output, "only.txt") {
			t.Errorf("should find matches in .txt files, got %q", result.Output)
		}
	})

	t.Run("skips unsupported file types", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "data.bin"), []byte("hidden_needle"), 0644)

		result, err := executeSearch(SearchArgs{Pattern: "hidden_needle"}, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Output != "No matches found." {
			t.Errorf("should not find matches in .bin files, got %q", result.Output)
		}
	})

	t.Run("relative path resolved against workingDir", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, "sub")
		os.MkdirAll(sub, 0755)
		os.WriteFile(filepath.Join(sub, "f.go"), []byte("rel_needle"), 0644)

		result, err := executeSearch(SearchArgs{Pattern: "rel_needle", Path: "sub"}, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result.Output, "rel_needle") {
			t.Errorf("should find match in relative subdir, got %q", result.Output)
		}
	})

	t.Run("truncates output at 30 matches", func(t *testing.T) {
		dir := t.TempDir()
		// Create a file with 35 matching lines
		var lines []string
		for i := 0; i < 35; i++ {
			lines = append(lines, fmt.Sprintf("match_line_%d", i))
		}
		os.WriteFile(filepath.Join(dir, "big.go"), []byte(strings.Join(lines, "\n")), 0644)

		result, err := executeSearch(SearchArgs{Pattern: "match_line_"}, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result.Output, "35 total matches, showing first 30") {
			t.Errorf("should indicate truncation, got %q", result.Output)
		}
		// Count output lines: 30 matches + 1 truncation message = 31
		outputLines := strings.Split(result.Output, "\n")
		if len(outputLines) != 31 {
			t.Errorf("expected 31 output lines (30 + truncation), got %d", len(outputLines))
		}
	})

	t.Run("truncates long lines at 200 chars", func(t *testing.T) {
		dir := t.TempDir()
		longLine := "needle_start_" + strings.Repeat("x", 300)
		os.WriteFile(filepath.Join(dir, "long.go"), []byte(longLine), 0644)

		result, err := executeSearch(SearchArgs{Pattern: "needle_start_"}, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Each line in output should be at most 203 chars (200 + "...")
		for _, line := range strings.Split(result.Output, "\n") {
			if len(line) > 203 {
				t.Errorf("line exceeds 200 char truncation limit: len=%d", len(line))
			}
		}
		if !strings.HasSuffix(result.Output, "...") {
			t.Errorf("truncated line should end with '...', got %q", result.Output)
		}
	})

	t.Run("excludes .git directory", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		os.MkdirAll(gitDir, 0755)
		os.WriteFile(filepath.Join(gitDir, "config.txt"), []byte("git_secret"), 0644)
		os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)

		result, err := executeSearch(SearchArgs{Pattern: "git_secret"}, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Output != "No matches found." {
			t.Errorf("should not search .git directory, got %q", result.Output)
		}
	})
}
