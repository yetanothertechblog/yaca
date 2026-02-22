package tui

import (
	"fmt"
	"strings"
)

func canGroupToolCall(entry ChatEntry) bool {
	if entry.Type != EntryToolCall || entry.Denied {
		return false
	}

	name, _ := splitCommand(entry.Command)
	return name == "read_file" || name == "list_files" || name == "search"
}

func isBlankAssistant(e ChatEntry) bool {
	return e.Type == EntryMessage &&
		strings.TrimSpace(e.Content) == "" &&
		strings.TrimSpace(e.ReasoningContent) == ""
}

func findGroupEnd(messages []ChatEntry, start int) int {
	end := start
	for i := start + 1; i < len(messages); i++ {
		entry := messages[i]
		if isBlankAssistant(entry) {
			continue
		}
		if entry.Type != EntryToolCall || entry.Denied || !canGroupToolCall(entry) {
			break
		}
		end = i
	}
	return end
}

func countOperations(group []ChatEntry) (reads, searches, lists int) {
	for _, entry := range group {
		name, _ := splitCommand(entry.Command)
		switch name {
		case "read_file":
			reads++
		case "search":
			searches++
		case "list_files":
			lists++
		}
	}
	return reads, searches, lists
}

func renderGroupedToolCalls(group []ChatEntry) string {
	var toolEntries []ChatEntry
	for _, e := range group {
		if canGroupToolCall(e) {
			toolEntries = append(toolEntries, e)
		}
	}

	reads, searches, lists := countOperations(toolEntries)

	var parts []string
	if reads > 0 {
		parts = append(parts, fmt.Sprintf("📖 Read %d files", reads))
	}
	if searches > 0 {
		parts = append(parts, fmt.Sprintf("🔍 Searched for %d patterns", searches))
	}
	if lists > 0 {
		parts = append(parts, fmt.Sprintf("📁 Listed %d directories", lists))
	}

	header := toolBulletStyle.Render("⏺ ") + toolCmdStyle.Render(strings.Join(parts, ", "))

	// Show last 3 tool call titles indented
	maxShown := 3
	var titles []string
	for i := len(toolEntries) - 1; i >= 0; i-- {
		if len(toolEntries)-i > maxShown {
			titles = append(titles, fmt.Sprintf("...%d more", len(toolEntries)-maxShown))
			break
		}
		titles = append(titles, formatCommand(toolEntries[i].Command))
	}

	return header + "\n" + indentBlock(strings.Join(titles, "\n"))
}
