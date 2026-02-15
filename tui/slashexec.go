package tui

import (
	"strings"

	"go-tui/tui/slashcmd"

	tea "github.com/charmbracelet/bubbletea"
)

// executeSlashCommand checks if text is a known slash command and executes it.
// Returns (true, cmd) if the text was handled as a command, (false, nil) otherwise.
func (m *Model) executeSlashCommand(text string) (bool, tea.Cmd) {
	switch text {
	case "/clear":
		m.messages = nil
		m.history = nil
		m.totalTokens = 0
		m.saveConversation()
		m.refreshViewport()
		return true, nil
	case "/compact":
		if len(m.history) == 0 {
			m.appendError("Nothing to compact")
			m.refreshViewport()
			return true, nil
		}
		m.waiting = true
		m.textarea.Blur()
		return true, compactHistory(m.history)
	case "/rewind":
		return m.executeRewind()
	case "/help", "/status":
		m.appendError("Command not yet implemented")
		m.refreshViewport()
		return true, nil
	default:
		return false, nil
	}
}

func (m *Model) executeRewind() (bool, tea.Cmd) {
	var items []slashcmd.RewindItem
	historyPos := 0
	for mi, entry := range m.messages {
		if entry.Role != "user" {
			continue
		}
		// Find matching history entry by content, scanning forward
		hi := -1
		for j := historyPos; j < len(m.history); j++ {
			if m.history[j].Role == "user" && m.history[j].Content == entry.Content {
				hi = j
				historyPos = j + 1
				break
			}
		}
		if hi == -1 {
			continue
		}

		display := entry.Content
		if len(display) > 60 {
			display = display[:57] + "..."
		}
		// Replace newlines with spaces for single-line display
		display = strings.ReplaceAll(display, "\n", " ")

		items = append(items, slashcmd.RewindItem{
			Text:         display,
			FullText:     entry.Content,
			MessageIndex: mi,
			HistoryIndex: hi,
		})
	}

	if len(items) == 0 {
		m.appendError("Nothing to rewind")
		m.refreshViewport()
		return true, nil
	}

	m.rewindOverlay = &slashcmd.RewindOverlay{
		Items:  items,
		Cursor: len(items) - 1,
	}
	return true, nil
}
