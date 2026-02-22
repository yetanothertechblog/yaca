package tui

import (
	"os"
	"path/filepath"
	"strings"

	"go-tui/config"
	"go-tui/conversation"
	"go-tui/tui/slashcmd"

	tea "github.com/charmbracelet/bubbletea"
)

// executeSlashCommand checks if text is a known slash command and executes it.
// Returns (true, cmd) if the text was handled as a command, (false, nil) otherwise.
func (m *Model) executeSlashCommand(text string) (bool, tea.Cmd) {
	switch text {
	case "/exit":
		return true, tea.Quit
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
	case "/resume":
		return m.executeResume()
	case "/model":
		return m.executeModel()
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

func (m *Model) executeResume() (bool, tea.Cmd) {
	dir := conversation.Dir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		m.appendError("No conversations found")
		m.refreshViewport()
		return true, nil
	}

	var items []slashcmd.RewindItem
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		preview, err := conversation.ReadPreview(path)
		if err != nil || preview.ID == m.conv.ID {
			continue
		}
		label := strings.ReplaceAll(preview.FirstMsg, "\n", " ")
		if len(label) > 60 {
			label = label[:57] + "..."
		}
		if label == "" {
			label = preview.ID
		}
		items = append(items, slashcmd.RewindItem{
			Text: label,
			Path: path,
		})
	}

	if len(items) == 0 {
		m.appendError("No other conversations to resume")
		m.refreshViewport()
		return true, nil
	}

	// Reverse so newest conversation is at the top
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}

	m.conversationOverlay = &slashcmd.RewindOverlay{
		Title:  "Resume conversation",
		Items:  items,
		Cursor: 0,
	}
	return true, nil
}

func (m *Model) executeModel() (bool, tea.Cmd) {
	names := config.ModelNames()
	active := m.settings.ActiveModel()
	cursor := 0
	for i, n := range names {
		if n == active {
			cursor = i
			break
		}
	}

	m.modelOverlay = &slashcmd.ModelOverlay{
		Items:  names,
		Cursor: cursor,
		Active: active,
	}
	return true, nil
}
