package tui

import (
	"fmt"
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
