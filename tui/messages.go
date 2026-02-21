package tui

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"go-tui/config"
)

// Styles are defined in theme.go

func renderMessages(messages []ChatEntry, perm *PermissionPrompt, width int, md *MarkdownRenderer) string {
	if len(messages) == 0 && perm == nil {
		return "Welcome! Type a message and press Enter to send."
	}

	var sb strings.Builder
	// Debug: log entry types to verify interleaving
	for idx, e := range messages {
		log.Printf("messages[%d]: Type=%d Role=%q Command=%q", idx, e.Type, e.Role, e.Command)
	}
	i := 0
	for i < len(messages) {
		entry := messages[i]

		var rendered string

		// Check if we can group this and next entries
		if entry.Type == EntryToolCall && canGroupToolCall(entry) && i+1 < len(messages) {
			groupEnd := findGroupEnd(messages, i)
			if groupEnd > i {
				rendered = renderGroupedToolCalls(messages[i : groupEnd+1])
				rendered = strings.Trim(rendered, "\n")
				sb.WriteString(rendered + "\n\n")
				i = groupEnd + 1
				continue
			}
		}

		// Render individual entry
		switch entry.Type {
		case EntryToolCall:
			rendered = renderToolCallEntry(entry)

		case EntryMessage:
			rendered = renderMessageEntry(entry, md)
		}

		if entry.Error != "" {
			rendered += "\n" + indentBlock(errorStyle.Render(entry.Error))
		}

		rendered = strings.Trim(rendered, "\n")
		sb.WriteString(rendered + "\n\n")

		i++
	}

	// Show permission prompt inline
	if perm != nil {
		if len(messages) > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(perm.View(width))
	}

	return sb.String()
}

func renderToolCallEntry(entry ChatEntry) string {
	header := formatCommand(entry.Command)
	bullet := toolBulletStyle.Render("⏺ ") + toolCmdStyle.Render(header)

	if entry.Denied {
		if entry.Diff != nil {
			return bullet + "\n" + indentBlock(renderDiff(*entry.Diff)) + "\n" + indentBlock(deniedStyle.Render("User declined"))
		}
		return bullet + " " + deniedStyle.Render("User declined")
	}

	if entry.Diff != nil {
		return bullet + "\n" + indentBlock(renderDiff(*entry.Diff))
	}

	name, _ := splitCommand(entry.Command)

	switch name {
	case "read_file":
		return bullet
	case "list_files", "bash":
		result := entry.Result
		lines := strings.Split(result, "\n")
		if len(lines) > 3 {
			result = strings.Join(lines[:3], "\n") + fmt.Sprintf("\n... (%d more lines)", len(lines)-3)
		}
		return bullet + "\n" + indentBlock(result)
	default:
		result := entry.Result
		maxResultLines := config.MaxResultLines
		lines := strings.Split(result, "\n")
		if len(lines) > maxResultLines {
			result = strings.Join(lines[:maxResultLines], "\n") + fmt.Sprintf("\n... (%d more lines)", len(lines)-maxResultLines)
		}
		return bullet + "\n" + indentBlock(result)
	}
}

func renderMessageEntry(entry ChatEntry, md *MarkdownRenderer) string {
	switch entry.Role {
	case "user":
		return userMessageStyle.Render("❯ " + entry.Content)
	case "assistant":
		return renderAssistantMessage(entry.Content, md)
	default:
		return fmt.Sprintf("%s: %s", entry.Role, entry.Content)
	}
}

func renderAssistantMessage(content string, md *MarkdownRenderer) string {
	if md != nil && isMarkdown(content) {
		if r, err := md.Render(content); err == nil {
			return r
		}
	}
	return content
}

// indentBlock renders content with ⎿ on the first line, then spaces for the rest.
func indentBlock(content string) string {
	content = strings.TrimRight(content, "\n ")
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return ""
	}
	bar := toolIndentStyle.Render("   ⎿ ")
	pad := "     "
	out := bar + lines[0]
	for _, line := range lines[1:] {
		out += "\n" + pad + line
	}
	return out
}

// Helper functions for formatCommand
func formatReadCommand(icon string, str func(string) string, num func(string) (int, bool)) string {
	s := "Read: " + str("file_path")
	offset, hasOffset := num("offset")
	limit, hasLimit := num("limit")
	if hasOffset && hasLimit {
		s += fmt.Sprintf(" %d:%d", offset, offset+limit-1)
	} else if hasOffset {
		s += fmt.Sprintf(" from %d", offset)
	} else if hasLimit {
		s += fmt.Sprintf(" first %d lines", limit)
	}
	return icon + s
}

func formatListCommand(icon string, str func(string) string) string {
	path := str("path")
	if path == "" {
		path = "."
	}
	return icon + "List: " + path
}

func formatSearchCommand(icon string, str func(string) string) string {
	s := "Search: " + str("pattern")
	if p := str("path"); p != "" {
		s += " in " + p
	}
	return icon + s
}

// formatCommand turns "tool_name: {json}" into a human-readable string.
func formatCommand(command string) string {
	name, argsJSON := splitCommand(command)
	if argsJSON == "" {
		return command
	}

	var args map[string]json.RawMessage
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return command
	}

	str := func(key string) string {
		raw, ok := args[key]
		if !ok {
			return ""
		}
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return s
		}
		return ""
	}

	num := func(key string) (int, bool) {
		raw, ok := args[key]
		if !ok {
			return 0, false
		}
		var n int
		if json.Unmarshal(raw, &n) == nil {
			return n, true
		}
		return 0, false
	}

	var icon string
	switch name {
	case "read_file":
		icon = config.ReadIcon
	case "list_files":
		icon = config.ListIcon
	case "bash":
		icon = config.BashIcon
	case "search":
		icon = config.SearchIcon
	case "edit_file":
		icon = config.EditIcon
	case "write_file":
		icon = config.WriteIcon
	default:
		icon = config.ToolIcon
	}

	switch name {
	case "read_file":
		return formatReadCommand(icon, str, num)
	case "list_files":
		return formatListCommand(icon, str)
	case "bash":
		return icon + "Bash: " + str("command")
	case "search":
		return formatSearchCommand(icon, str)
	case "edit_file":
		return icon + "Edit: " + str("file_path")
	case "write_file":
		return icon + "Write: " + str("file_path")
	default:
		return icon + command
	}
}

// splitCommand splits "tool_name: {json}" into name and argsJSON.
func splitCommand(command string) (string, string) {
	idx := strings.Index(command, ": ")
	if idx < 0 {
		return command, ""
	}
	return command[:idx], command[idx+2:]
}
