// mdtest renders a set of incomplete markdown snippets through glamour to
// reveal how partial streaming content looks in the terminal.
package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
)

func render(label, input string) {
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		panic(err)
	}
	out, err := r.Render(input)
	errStr := "nil"
	if err != nil {
		errStr = err.Error()
	}
	sep := strings.Repeat("─", 80)
	fmt.Printf("\n%s\n[%s]\nINPUT  : %q\nerr    : %s\nOUTPUT :\n%s%s\n",
		sep, label, input, errStr, out, sep)
}

func sanitize(s string) string {
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

func main() {
	cases := []struct {
		label string
		input string
	}{
		// Triple-backtick fence
		{"open triple-backtick (no lang)", "Some text\n```\nfunc hello() {"},
		{"open triple-backtick with lang", "Here is code:\n```go\nfunc hello() {"},
		{"closed triple-backtick", "Here is code:\n```go\nfunc hello() {}\n```"},

		// Single backtick inline code
		{"open single backtick", "Use `os.Exit to quit"},
		{"closed single backtick", "Use `os.Exit` to quit"},

		// Heading dangling (mid-word)
		{"heading mid-word", "# Hel"},
		{"heading complete", "# Hello World"},
		{"heading no space after #", "#Hello"},

		// Bold / italic
		{"open bold **", "This is **important"},
		{"closed bold **", "This is **important**"},
		{"open italic *", "This is *important"},
		{"open italic _", "This is _important"},

		// Blockquote
		{"open blockquote", "> This is a partial"},
		{"multi-line blockquote unclosed", "> line one\n> line two\nplain"},

		// List items
		{"incomplete list item", "- item one\n- item tw"},
		{"incomplete ordered list", "1. first\n2. seco"},

		// Link
		{"open link text", "See [this"},
		{"open link url", "See [this](https://exam"},
		{"closed link", "See [this](https://example.com)"},

		// Table
		{"incomplete table header", "| col1 | col2 |\n| --- |"},
		{"incomplete table row", "| col1 | col2 |\n| --- | --- |\n| val1 |"},

		// Mixed: prose then unclosed fence
		{"prose then open fence", "Here is the plan:\n\n1. Step one\n2. Step two\n\n```python\ndef foo():"},

		// Sanitizer: blockquote -> plain text (the one bad case, before and after fix)
		{"blockquote+plain UNSANITIZED", "> line one\n> line two\nplain text here"},
		{"blockquote+plain SANITIZED", sanitize("> line one\n> line two\nplain text here")},
	}

	for _, c := range cases {
		render(c.label, c.input)
	}
}
