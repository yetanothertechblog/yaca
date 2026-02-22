package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// Styles are defined in theme.go

// renderDiff renders a unified-style diff with colored +/- lines and line numbers.
// Deletions are grouped before additions in each change hunk.
// When OldText is empty, all lines are shown as additions (new file).
func renderDiff(d DiffData) string {
	var sb strings.Builder

	if d.OldText == "" {
		for i, line := range strings.Split(d.NewText, "\n") {
			num := diffAddedLineNumStyle.Render(fmt.Sprintf("   %4d ", i+1))
			marker := diffAddedMarkerStyle.Render("+")
			content := diffAddedStyle.Render(" " + line)
			sb.WriteString(num + marker + content + "\n")
		}
		return strings.TrimRight(sb.String(), "\n")
	}

	startLine := d.StartLine
	if startLine < 1 {
		startLine = 1
	}
	oldLine := startLine
	newLine := startLine

	dmp := diffmatchpatch.New()
	a, b, lines := dmp.DiffLinesToChars(d.OldText, d.NewText)
	diffs := dmp.DiffMain(a, b, false)
	diffs = dmp.DiffCharsToLines(diffs, lines)
	diffs = dmp.DiffCleanupEfficiency(diffs)

	// Collect diffs into hunks: group consecutive delete+insert together,
	// rendering all deletions before all additions in each hunk.
	i := 0
	for i < len(diffs) {
		diff := diffs[i]

		if diff.Type == diffmatchpatch.DiffEqual {
			text := strings.TrimRight(diff.Text, "\n")
			for _, line := range strings.Split(text, "\n") {
				num := diffLineNumStyle.Render(fmt.Sprintf("%4d ", newLine))
				sb.WriteString(num + diffHunkStyle.Render("  "+line))
				sb.WriteString("\n")
				oldLine++
				newLine++
			}
			i++
			continue
		}

		// Collect consecutive delete and insert chunks as one hunk
		var deleted, added []string
		for i < len(diffs) && diffs[i].Type != diffmatchpatch.DiffEqual {
			text := strings.TrimRight(diffs[i].Text, "\n")
			lines := strings.Split(text, "\n")
			if diffs[i].Type == diffmatchpatch.DiffDelete {
				deleted = append(deleted, lines...)
			} else {
				added = append(added, lines...)
			}
			i++
		}

		// Render all deletions first
		for _, line := range deleted {
			num := diffRemovedLineNumStyle.Render(fmt.Sprintf("%4d ", oldLine))
			marker := diffRemovedMarkerStyle.Render("-")
			content := diffRemovedStyle.Render(" " + line)
			sb.WriteString(num + marker + content + "\n")
			oldLine++
		}
		// Then all additions
		for _, line := range added {
			num := diffAddedLineNumStyle.Render(fmt.Sprintf("%4d ", newLine))
			marker := diffAddedMarkerStyle.Render("+")
			content := diffAddedStyle.Render(" " + line)
			sb.WriteString(num + marker + content + "\n")
			newLine++
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

// findStartLine returns the 1-based line number where needle starts within fileContent.
// Returns 1 if not found.
func findStartLine(fileContent, needle string) int {
	idx := strings.Index(fileContent, needle)
	if idx < 0 {
		return 1
	}
	return strings.Count(fileContent[:idx], "\n") + 1
}

// getDiffForPermission renders a diff preview for the permission prompt.
func getDiffForPermission(toolName, argsJSON, workingDir string) string {
	d := parseDiffFromArgs(toolName, argsJSON, workingDir)
	if d == nil {
		return ""
	}
	return renderDiff(*d)
}

func parseDiffFromToolCall(toolName, args, result, workingDir string, denied bool) *DiffData {
	if denied {
		return parseDiffFromArgs(toolName, args, workingDir)
	}

	if result == "" {
		return nil
	}

	switch toolName {
	case "write_file":
		var r struct {
			FilePath   string `json:"file_path"`
			NewContent string `json:"new_content"`
		}
		if json.Unmarshal([]byte(result), &r) != nil || r.FilePath == "" {
			return parseDiffFromArgs(toolName, args, workingDir)
		}
		return &DiffData{
			FilePath: r.FilePath,
			NewText:  r.NewContent,
		}
	case "edit_file":
		var r struct {
			FilePath  string `json:"file_path"`
			OldString string `json:"old_string"`
			NewString string `json:"new_string"`
		}
		if json.Unmarshal([]byte(result), &r) != nil || r.FilePath == "" {
			return parseDiffFromArgs(toolName, args, workingDir)
		}
		startLine := 1
		path := r.FilePath
		if !filepath.IsAbs(path) {
			path = filepath.Join(workingDir, path)
		}
		if data, err := os.ReadFile(path); err == nil {
			startLine = findStartLine(string(data), r.OldString)
		}
		return &DiffData{
			FilePath:  r.FilePath,
			OldText:   r.OldString,
			NewText:   r.NewString,
			StartLine: startLine,
		}
	}
	return nil
}

func parseDiffFromArgs(name, argsJSON, workingDir string) *DiffData {
	switch name {
	case "edit_file":
		var args struct {
			FilePath  string `json:"file_path"`
			OldString string `json:"old_string"`
			NewString string `json:"new_string"`
		}
		if json.Unmarshal([]byte(argsJSON), &args) != nil || args.FilePath == "" {
			return nil
		}
		startLine := 1
		path := args.FilePath
		if !filepath.IsAbs(path) {
			path = filepath.Join(workingDir, path)
		}
		if data, err := os.ReadFile(path); err == nil {
			startLine = findStartLine(string(data), args.OldString)
		}
		return &DiffData{
			FilePath:  args.FilePath,
			OldText:   args.OldString,
			NewText:   args.NewString,
			StartLine: startLine,
		}

	case "write_file":
		var args struct {
			FilePath string `json:"file_path"`
			Content  string `json:"content"`
		}
		if json.Unmarshal([]byte(argsJSON), &args) != nil || args.FilePath == "" {
			return nil
		}
		return &DiffData{
			FilePath: args.FilePath,
			NewText:  args.Content,
		}
	}
	return nil
}
