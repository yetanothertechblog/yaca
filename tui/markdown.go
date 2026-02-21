package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
)

// MarkdownRenderer handles markdown rendering for the TUI.
type MarkdownRenderer struct {
	renderer *glamour.TermRenderer
}

// NewMarkdownRenderer creates a new markdown renderer.
func NewMarkdownRenderer(width int) (*MarkdownRenderer, error) {
	style := styles.DarkStyleConfig
	noMargin := uint(0)
	style.Document.Margin = &noMargin
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(style),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}
	return &MarkdownRenderer{renderer: renderer}, nil
}

// Render converts markdown content to styled terminal text.
func (r *MarkdownRenderer) Render(markdown string) (string, error) {
	if r == nil || r.renderer == nil {
		return markdown, nil
	}
	rendered, err := r.renderer.Render(markdown)
	if err != nil {
		return markdown, err
	}
	return strings.Trim(rendered, "\n"), nil
}

// sanitizePartialMarkdown fixes known Goldmark edge cases that arise with
// incomplete streaming content. Currently handles one case:
//
//	Blockquote lines followed immediately by a non-blockquote line without a
//	blank line separator — Goldmark absorbs the plain line into the blockquote.
//	Inserting a blank line between them restores the correct parse.
func sanitizePartialMarkdown(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines)+4)
	for i, line := range lines {
		if i > 0 {
			prev := lines[i-1]
			if strings.HasPrefix(prev, "> ") && !strings.HasPrefix(line, "> ") && line != "" {
				out = append(out, "")
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// renderStreamingMarkdown renders partial markdown content through glamour,
// falling back to plain text if rendering fails.
// It skips the isMarkdown heuristic — always attempts glamour so styled
// content appears as soon as the first token arrives.
func renderStreamingMarkdown(content string, md *MarkdownRenderer) string {
	if md == nil {
		return strings.Trim(content, "\n")
	}
	sanitized := sanitizePartialMarkdown(content)
	if r, err := md.Render(sanitized); err == nil {
		return r
	}
	return strings.Trim(content, "\n")
}

// isMarkdown returns true if the content likely contains markdown formatting.
func isMarkdown(content string) bool {
	markers := []string{
		"```", "# ", "## ", "### ",
		"**", "* ", "- ", "| ",
		"1. ", "> ",
	}
	for _, m := range markers {
		if strings.Contains(content, m) {
			return true
		}
	}
	return false
}
