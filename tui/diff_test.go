package tui

import (
	"regexp"
	"strings"
	"testing"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

// diffLines returns all non-empty lines from the stripped diff output.
func diffLines(out string) []string {
	var result []string
	for _, l := range strings.Split(stripANSI(out), "\n") {
		if t := strings.TrimSpace(l); t != "" {
			result = append(result, t)
		}
	}
	return result
}

// hasLine checks whether any output line contains the given substring.
func hasLine(lines []string, substr string) bool {
	for _, l := range lines {
		if strings.Contains(l, substr) {
			return true
		}
	}
	return false
}

// --- BlockReplace bug ---

// TestBlockReplaceDuplicatesUnchangedLines documents the current bug:
// BlockReplace renders every old line as "-" and every new line as "+",
// so unchanged lines appear as both "- line" and "+ line".
func TestBlockReplaceDuplicatesUnchangedLines(t *testing.T) {
	d := DiffData{
		FilePath:  "foo.go",
		OldText:   "line1\nchanged\nline3",
		NewText:   "line1\nnew_content\nline3",
		StartLine: 1,
	}
	lines := diffLines(renderDiff(d))

	hasMinus := hasLine(lines, "- line1")
	hasPlus := hasLine(lines, "+ line1")
	if hasMinus && hasPlus {
		t.Error("BUG: unchanged line1 appears as both deletion and addition with BlockReplace")
	}
}

// --- Desired behaviour after fix ---

// TestRenderDiffDoesNotDuplicateUnchangedLines is the correctness test:
// unchanged lines must not appear as +/- markers.
func TestRenderDiffDoesNotDuplicateUnchangedLines(t *testing.T) {
	d := DiffData{
		FilePath:  "foo.go",
		OldText:   "line1\nchanged\nline3",
		NewText:   "line1\nnew_content\nline3",
		StartLine: 1,
	}
	lines := diffLines(renderDiff(d))

	if hasLine(lines, "- line1") {
		t.Error("unchanged line1 should not appear as deletion")
	}
	if hasLine(lines, "+ line1") {
		t.Error("unchanged line1 should not appear as addition")
	}
	if hasLine(lines, "- line3") {
		t.Error("unchanged line3 should not appear as deletion")
	}
	if hasLine(lines, "+ line3") {
		t.Error("unchanged line3 should not appear as addition")
	}
}

// TestRenderDiffShowsActualChanges verifies the changed line is rendered correctly.
func TestRenderDiffShowsActualChanges(t *testing.T) {
	d := DiffData{
		FilePath:  "foo.go",
		OldText:   "line1\nchanged\nline3",
		NewText:   "line1\nnew_content\nline3",
		StartLine: 1,
	}
	lines := diffLines(renderDiff(d))

	if !hasLine(lines, "- changed") {
		t.Error("expected old line to appear as deletion")
	}
	if !hasLine(lines, "+ new_content") {
		t.Error("expected new line to appear as addition")
	}
}

// TestRenderDiffNewFileShowsAllAdditions verifies the no-OldText path.
func TestRenderDiffNewFileShowsAllAdditions(t *testing.T) {
	d := DiffData{
		FilePath: "new.go",
		NewText:  "package main\nfunc main() {}",
	}
	lines := diffLines(renderDiff(d))

	if !hasLine(lines, "+ package main") {
		t.Error("expected first line to appear as addition")
	}
	if !hasLine(lines, "+ func main() {}") {
		t.Error("expected second line to appear as addition")
	}
	if hasLine(lines, "- ") {
		t.Error("no deletions expected for new file")
	}
}

// TestRenderDiffOnlyChangedBlockAtEnd verifies changes at the end of the snippet.
func TestRenderDiffOnlyChangedBlockAtEnd(t *testing.T) {
	d := DiffData{
		FilePath:  "foo.go",
		OldText:   "func foo() {\n\treturn 1\n}",
		NewText:   "func foo() {\n\treturn 2\n}",
		StartLine: 10,
	}
	lines := diffLines(renderDiff(d))

	if hasLine(lines, "- func foo()") {
		t.Error("unchanged func signature should not appear as deletion")
	}
	if hasLine(lines, "+ func foo()") {
		t.Error("unchanged func signature should not appear as addition")
	}
	// Tab characters are expanded to spaces by lipgloss, so check marker and
	// content independently on the same line.
	hasdeletion := func(content string) bool {
		for _, l := range lines {
			if strings.Contains(l, "- ") && strings.Contains(l, content) {
				return true
			}
		}
		return false
	}
	hasaddition := func(content string) bool {
		for _, l := range lines {
			if strings.Contains(l, "+ ") && strings.Contains(l, content) {
				return true
			}
		}
		return false
	}
	if !hasdeletion("return 1") {
		t.Error("expected old return to appear as deletion")
	}
	if !hasaddition("return 2") {
		t.Error("expected new return to appear as addition")
	}
}
