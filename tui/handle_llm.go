package tui

import (
	"log"
	"strings"

	"go-tui/llm"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleStreamChunk(msg StreamChunkMsg) (tea.Model, tea.Cmd) {
	m.streamingThinking = msg.Thinking
	m.streamingTokens += int(float64(len(strings.Fields(msg.Content))) * 0.75)
	m.streamingContent += msg.Content
	m.streamingThinkingContent += msg.ThinkingContent
	m.refreshViewport()
	return m, waitForStreamInterruptible(msg.ch, m.interruptCh)
}

func (m *Model) handleLLMResponse(msg LLMResponseMsg) (tea.Model, tea.Cmd) {
	m.streamingContent = ""
	m.streamingThinkingContent = ""
	m.streamingTokens = 0
	m.streamingThinking = false
	if msg.Usage != nil {
		m.totalTokens = msg.Usage.TotalTokens
	}
	if msg.Err != nil {
		m.waiting = false
		m.textarea.Focus()
		m.appendError(msg.Err.Error())
		m.saveConversation()
		m.refreshViewport()
		return m, nil
	}

	if len(msg.ToolCalls) == 0 {
		// No tools — plain assistant response
		m.waiting = false
		m.textarea.Focus()
		m.history = append(m.history, llm.Message{
			Role:    "assistant",
			Content: msg.Content,
		})
		if strings.TrimSpace(msg.Content) != "" || msg.ReasoningContent != "" {
			m.messages = append(m.messages, ChatEntry{
				Type:             EntryMessage,
				Role:             "assistant",
				Content:          msg.Content,
				ReasoningContent: msg.ReasoningContent,
			})
		}
		m.saveConversation()
		m.refreshViewport()
		return m, nil
	}

	// Has tool calls — append assistant message with both content and tool calls
	m.history = append(m.history, llm.Message{
		Role:      "assistant",
		Content:   msg.Content,
		ToolCalls: msg.ToolCalls,
	})

	// If there's meaningful content alongside tool calls, show it
	if strings.TrimSpace(msg.Content) != "" || msg.ReasoningContent != "" {
		m.messages = append(m.messages, ChatEntry{
			Type:             EntryMessage,
			Role:             "assistant",
			Content:          msg.Content,
			ReasoningContent: msg.ReasoningContent,
		})
	}

	m.pendingToolCalls = msg.ToolCalls
	m.pendingToolIndex = 0
	m.toolRoundCount++
	log.Printf("LLM round %d: %d tool calls, content=%q", m.toolRoundCount, len(msg.ToolCalls), msg.Content)
	m.refreshViewport()

	cmd := m.dispatchNextTool()
	return m, cmd
}

func (m *Model) handleCompactResult(msg CompactResultMsg) (tea.Model, tea.Cmd) {
	m.waiting = false
	m.textarea.Focus()
	if msg.Err != nil {
		m.appendError("Compact failed: " + msg.Err.Error())
		m.refreshViewport()
		return m, nil
	}
	m.history = []llm.Message{
		{
			Role:    "user",
			Content: "[Conversation summary]\n" + msg.Summary,
		},
	}
	m.messages = []ChatEntry{
		{
			Type:    EntryMessage,
			Role:    "assistant",
			Content: "Conversation compacted:\n\n" + msg.Summary,
		},
	}
	if msg.Usage != nil {
		m.totalTokens = msg.Usage.TotalTokens
	}
	m.saveConversation()
	m.refreshViewport()
	return m, nil
}
