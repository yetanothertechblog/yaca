package tui

import (
	"go-tui/llm"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleUserInput(msg UserInputMsg) (tea.Model, tea.Cmd) {
	text := msg.Text

	m.messages = append(m.messages, ChatEntry{
		Type:    EntryMessage,
		Role:    "user",
		Content: text,
	})

	// Append user message to history (now on Model, not Agent)
	m.history = append(m.history, llm.Message{
		Role:    "user",
		Content: text,
	})

	m.textarea.Reset()
	m.textarea.Blur()
	m.waiting = true
	m.toolRoundCount = 0
	m.consecutiveErrors = 0
	m.interruptCh = make(chan struct{})

	m.refreshViewport()

	return m, callLLMInterruptible(m.agent, m.history, m.interruptCh)
}

func (m *Model) handleInterrupt(msg InterruptMsg) (tea.Model, tea.Cmd) {
	m.waiting = false
	m.textarea.Focus()
	m.appendError(msg.Reason)
	m.saveConversation()
	m.refreshViewport()
	return m, nil
}

func (m *Model) handleRewind(msg RewindToMessageMsg) (tea.Model, tea.Cmd) {
	// Truncate messages and history to before the selected message
	m.messages = m.messages[:msg.MessageIndex]
	m.history = m.history[:msg.HistoryIndex]

	// Populate textarea with the selected message text
	m.textarea.SetValue(msg.FullText)

	m.saveConversation()
	m.refreshViewport()
	return m, nil
}
