package tui

import (
	"log"
	"strings"

	"go-tui/llm"
	"go-tui/permissions"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleToolResult(msg ToolResultMsg) (tea.Model, tea.Cmd) {
	command := msg.ToolName + ": " + msg.Args
	resultStr := msg.Result

	// Limit list_files output to 3 lines for display
	if msg.ToolName == "list_files" {
		lines := strings.Split(msg.Result, "\n")
		if len(lines) > 4 { // 3 lines + empty line
			resultStr = strings.Join(lines[:4], "\n") + "\n... (showing first 3 entries)"
		}
	}

	if msg.Err != nil {
		m.consecutiveErrors++
		if m.consecutiveErrors >= maxConsecutiveErrors {
			resultStr += " (Too many consecutive errors. Stop retrying and tell the user what went wrong.)"
		}
	} else {
		m.consecutiveErrors = 0
	}

	// Append tool result to history
	m.history = append(m.history, llm.Message{
		Role:       "tool",
		Content:    resultStr,
		ToolCallID: msg.ToolCallID,
	})

	// Append tool call entry to UI messages
	entry := ChatEntry{
		Type:    EntryToolCall,
		Command: command,
		Result:  msg.Result,
		Diff:    parseDiffFromToolCall(msg.ToolName, msg.Args, msg.Result, m.workingDir, false),
	}
	m.messages = append(m.messages, entry)
	m.saveConversation()
	m.refreshViewport()

	// Advance to next tool
	m.pendingToolIndex++
	cmd := m.dispatchNextTool()
	return m, cmd
}

func (m *Model) handlePermissionDecision(msg PermissionDecisionMsg) (tea.Model, tea.Cmd) {
	switch msg.Decision {
	case PermissionAllow:
		return m, executeToolInterruptible(m.agent, msg.ToolCall, m.interruptCh)

	case PermissionAlwaysAllow:
		entry := msg.ToolCall.Function.Name
		if entry == "bash" {
			if prefix := permissions.BashCommandPrefix(msg.ToolCall.Function.Arguments); prefix != "" {
				entry = permissions.BashEntry(prefix)
			}
		}
		if err := m.permissions.Add(entry); err != nil {
			log.Printf("failed to save permission: %v", err)
		}
		return m, executeToolInterruptible(m.agent, msg.ToolCall, m.interruptCh)

	case PermissionDeny:
		command := msg.ToolCall.Function.Name + ": " + msg.ToolCall.Function.Arguments
		result := "Tool call denied by user."

		// Append denial to history
		m.history = append(m.history, llm.Message{
			Role:       "tool",
			Content:    result,
			ToolCallID: msg.ToolCall.ID,
		})

		// Append denied tool call to UI messages
		m.messages = append(m.messages, ChatEntry{
			Type:    EntryToolCall,
			Command: command,
			Denied:  true,
			Diff:    parseDiffFromToolCall(msg.ToolCall.Function.Name, msg.ToolCall.Function.Arguments, "", m.workingDir, true),
		})

		// Stop the loop — return to user input
		m.pendingToolCalls = nil
		m.pendingToolIndex = 0
		m.waiting = false
		m.textarea.Focus()
		m.saveConversation()
		m.refreshViewport()
		return m, nil
	}

	return m, nil
}

// dispatchNextTool returns a Cmd to execute the next pending tool call,
// or starts the next LLM round if all tools are done.
func (m *Model) dispatchNextTool() tea.Cmd {
	if m.pendingToolIndex >= len(m.pendingToolCalls) {
		// All tools done for this round
		m.pendingToolCalls = nil
		m.pendingToolIndex = 0
		if m.toolRoundCount >= maxToolRounds {
			m.waiting = false
			m.textarea.Focus()
			m.appendError("Tool call limit reached")
			m.saveConversation()
			m.refreshViewport()
			return nil
		}
		// Start next LLM round
		return callLLMInterruptible(m.agent, m.history, m.interruptCh)
	}

	tc := m.pendingToolCalls[m.pendingToolIndex]

	if m.bypassPermissions || m.permissions.IsAllowed(tc.Function.Name, tc.Function.Arguments) {
		return executeToolInterruptible(m.agent, tc, m.interruptCh)
	}

	// Need permission
	m.awaitingPermission = &tc
	m.permission = &PermissionPrompt{
		ToolName:   tc.Function.Name,
		Args:       tc.Function.Arguments,
		Cursor:     0,
		WorkingDir: m.workingDir,
	}
	m.refreshViewport()
	return nil
}
