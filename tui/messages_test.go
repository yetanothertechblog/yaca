package tui

import (
	"strings"
	"testing"
)

// bashEntry builds a basic non-groupable tool call entry (bash).
func bashEntry(result string) ChatEntry {
	return ChatEntry{
		Type:    EntryToolCall,
		Command: `bash: {"command":"echo hi"}`,
		Result:  result,
	}
}

// readEntry builds a groupable read_file tool call entry.
func readEntry(path string) ChatEntry {
	return ChatEntry{
		Type:    EntryToolCall,
		Command: `read_file: {"file_path":"` + path + `"}`,
		Result:  "file contents",
	}
}

// assistantEntry builds an assistant message entry, optionally with reasoning.
func assistantEntry(content, reasoning string) ChatEntry {
	return ChatEntry{
		Type:             EntryMessage,
		Role:             "assistant",
		Content:          content,
		ReasoningContent: reasoning,
	}
}

// maxConsecutiveNewlines returns the longest run of '\n' characters found in s.
func maxConsecutiveNewlines(s string) int {
	max, cur := 0, 0
	for _, c := range s {
		if c == '\n' {
			cur++
			if cur > max {
				max = cur
			}
		} else {
			cur = 0
		}
	}
	return max
}

// assertNoExcessSpacing fails if the stripped output contains more than two
// consecutive newlines (i.e. more than one blank line) anywhere.
func assertNoExcessSpacing(t *testing.T, label, output string) {
	t.Helper()
	plain := stripANSI(output)
	if n := maxConsecutiveNewlines(plain); n > 2 {
		t.Errorf("%s: found %d consecutive newlines (want ≤2):\n%q", label, n, plain)
	}
}

func TestToolCallSpacing(t *testing.T) {
	tests := []struct {
		name     string
		messages []ChatEntry
	}{
		{
			name: "two bash calls with results",
			messages: []ChatEntry{
				bashEntry("line1\nline2"),
				bashEntry("other output"),
			},
		},
		{
			name: "two bash calls with empty results",
			messages: []ChatEntry{
				bashEntry(""),
				bashEntry(""),
			},
		},
		{
			name: "three bash calls",
			messages: []ChatEntry{
				bashEntry("a"),
				bashEntry("b"),
				bashEntry("c"),
			},
		},
		{
			name: "bash then read_file",
			messages: []ChatEntry{
				bashEntry("output"),
				readEntry("foo.go"),
			},
		},
		{
			name: "read_file then bash (no group formed)",
			messages: []ChatEntry{
				readEntry("foo.go"),
				bashEntry("output"),
			},
		},
		{
			name: "two read_files then bash",
			messages: []ChatEntry{
				readEntry("a.go"),
				readEntry("b.go"),
				bashEntry("output"),
			},
		},
		{
			name: "bash between two read_files",
			messages: []ChatEntry{
				readEntry("a.go"),
				bashEntry("output"),
				readEntry("b.go"),
			},
		},
		// Assistant entries with empty/whitespace-only thinking content
		// sandwiched between tool calls — these are a common source of extra blank lines.
		{
			name: "bash, empty-thinking assistant, bash",
			messages: []ChatEntry{
				bashEntry("output"),
				assistantEntry("", ""),
				bashEntry("output"),
			},
		},
		{
			name: "bash, newline-only thinking assistant, bash",
			messages: []ChatEntry{
				bashEntry("output"),
				assistantEntry("", "\n"),
				bashEntry("output"),
			},
		},
		{
			name: "bash, whitespace thinking assistant, bash",
			messages: []ChatEntry{
				bashEntry("output"),
				assistantEntry("", "   \n  \n"),
				bashEntry("output"),
			},
		},
		{
			name: "multiple tool calls with empty-thinking assistants between each",
			messages: []ChatEntry{
				bashEntry("a"),
				assistantEntry("", ""),
				bashEntry("b"),
				assistantEntry("", "\n"),
				bashEntry("c"),
			},
		},
		{
			name: "read_files, empty-thinking assistant, bash",
			messages: []ChatEntry{
				readEntry("a.go"),
				readEntry("b.go"),
				assistantEntry("", ""),
				bashEntry("output"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := renderMessages(tc.messages, nil, 80, nil, "", "")
			assertNoExcessSpacing(t, tc.name, out)
		})
	}
}

func TestGroupingWithBlankAssistants(t *testing.T) {
	// Two read_files with a blank assistant in between should still be grouped
	// into a single "Read 2 files" entry, not rendered as separate bullets.
	t.Run("two reads separated by blank assistant are grouped", func(t *testing.T) {
		messages := []ChatEntry{
			readEntry("a.go"),
			assistantEntry("", ""),
			readEntry("b.go"),
		}
		out := stripANSI(renderMessages(messages, nil, 80, nil, "", ""))
		if strings.Count(out, "Read 2 files") != 1 {
			t.Errorf("expected grouped 'Read 2 files', got:\n%q", out)
		}
	})

	t.Run("three reads separated by blank assistants are grouped", func(t *testing.T) {
		messages := []ChatEntry{
			readEntry("a.go"),
			assistantEntry("", "\n"),
			readEntry("b.go"),
			assistantEntry("", ""),
			readEntry("c.go"),
		}
		out := stripANSI(renderMessages(messages, nil, 80, nil, "", ""))
		if strings.Count(out, "Read 3 files") != 1 {
			t.Errorf("expected grouped 'Read 3 files', got:\n%q", out)
		}
	})
}
