package tui

import (
	"encoding/json"

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
	m.streamingContent = ""
	m.streamingThinkingContent = ""
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

func (m *Model) handleModelSwitched(msg ModelSwitchedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.appendError("Failed to switch model: " + msg.Err.Error())
	} else {
		m.messages = append(m.messages, ChatEntry{
			Type:    EntryMessage,
			Role:    "assistant",
			Content: "Switched to model: " + msg.Name,
		})
	}
	m.refreshViewport()
	return m, nil
}

func (m *Model) handleResumeConversation(msg ResumeConversationMsg) (tea.Model, tea.Cmd) {
	var messages []ChatEntry
	if err := json.Unmarshal(msg.Conv.UIMessages, &messages); err != nil {
		m.appendError("Failed to restore conversation: " + err.Error())
		m.refreshViewport()
		return m, nil
	}
	var history []llm.Message
	if err := json.Unmarshal(msg.Conv.AgentHistory, &history); err != nil {
		m.appendError("Failed to restore conversation history: " + err.Error())
		m.refreshViewport()
		return m, nil
	}
	m.messages = messages
	m.history = history
	m.conv = msg.Conv
	m.totalTokens = 0
	m.toolRoundCount = 0
	m.consecutiveErrors = 0
	m.pendingToolCalls = nil
	m.pendingToolIndex = 0
	m.waiting = false
	m.textarea.Focus()
	m.refreshViewport()
	return m, nil
}
