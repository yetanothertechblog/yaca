package tui

import (
	"strings"
	"time"

	"go-tui/llm"
	"go-tui/tui/slashcmd"

	tea "github.com/charmbracelet/bubbletea"
)

func handleKeyMsg(m *Model, msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	}

	// Handle double ESC for interrupting long-running operations
	if msg.Type == tea.KeyEsc {
		now := time.Now()
		if now.Sub(m.lastEscTime) < 500*time.Millisecond {
			if m.waiting && m.interruptCh != nil {
				close(m.interruptCh)
				m.interruptCh = nil
				m.lastEscTime = time.Time{}
				return m, nil
			}
		}
		m.lastEscTime = now
		return m, nil
	}

	// Permission prompt mode
	if m.permission != nil {
		return handlePermissionKey(m, msg)
	}

	// Rewind overlay mode
	if m.rewindOverlay != nil {
		return handleRewindOverlayKey(m, msg)
	}

	// Slash overlay mode
	if m.slashOverlay != nil {
		return handleSlashOverlayKey(m, msg)
	}

	// Handle scrolling keys before textarea consumes them (works during waiting too)
	switch msg.Type {
	case tea.KeyPgUp:
		m.viewport.ViewUp()
		return m, nil
	case tea.KeyPgDown:
		m.viewport.ViewDown()
		return m, nil
	}

	switch msg.Type {
	case tea.KeyEnter:
		if m.waiting {
			return m, nil
		}

		text := strings.TrimSpace(m.textarea.Value())
		if text == "" {
			return m, nil
		}

		if handled, cmd := m.executeSlashCommand(text); handled {
			m.textarea.Reset()
			return m, cmd
		}

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

	if m.waiting {
		return m, nil
	}

	// Drop leaked CSI mouse sequence fragments (e.g. "[<64;32;32M", "<64;32;32M",
	// "32;32M") that arrive as KeyRunes when bubbletea's mouse parser doesn't
	// fully consume them during rapid scrolling.
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 1 {
		isMouse := true
		for _, r := range msg.Runes {
			if !((r >= '0' && r <= '9') || r == ';' || r == '[' || r == '<' || r == '>' || r == 'M' || r == 'm') {
				isMouse = false
				break
			}
		}
		if isMouse {
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)

	// Check if textarea now starts with "/" to open overlay
	m.updateSlashOverlay()

	return m, cmd
}

func handleSlashOverlayKey(m *Model, msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.slashOverlay.Cursor > 0 {
			m.slashOverlay.Cursor--
		}
		return m, nil

	case tea.KeyDown:
		if m.slashOverlay.Cursor < len(m.slashOverlay.Commands)-1 {
			m.slashOverlay.Cursor++
		}
		return m, nil

	case tea.KeyEnter, tea.KeyTab:
		if len(m.slashOverlay.Commands) > 0 {
			selected := m.slashOverlay.Commands[m.slashOverlay.Cursor]
			m.slashOverlay = nil
			_, cmd := m.executeSlashCommand(selected.Name)
			m.textarea.Reset()
			return m, cmd
		}
		m.slashOverlay = nil
		return m, nil

	case tea.KeyEsc:
		m.slashOverlay = nil
		return m, nil
	}

	// Pass other keys to textarea, then re-evaluate
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	m.updateSlashOverlay()

	return m, cmd
}

// updateSlashOverlay opens, updates, or closes the slash overlay based on textarea content.
func (m *Model) updateSlashOverlay() {
	val := m.textarea.Value()
	if !strings.HasPrefix(val, "/") {
		m.slashOverlay = nil
		return
	}

	filtered := slashcmd.Filter(val)
	if len(filtered) == 0 {
		m.slashOverlay = nil
		return
	}

	cursor := 0
	if m.slashOverlay != nil {
		cursor = m.slashOverlay.Cursor
		if cursor >= len(filtered) {
			cursor = len(filtered) - 1
		}
	}

	m.slashOverlay = &slashcmd.Overlay{
		Commands: filtered,
		Cursor:   cursor,
	}
}

func handleRewindOverlayKey(m *Model, msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.rewindOverlay.Cursor > 0 {
			m.rewindOverlay.Cursor--
		}
		return m, nil

	case tea.KeyDown:
		if m.rewindOverlay.Cursor < len(m.rewindOverlay.Items)-1 {
			m.rewindOverlay.Cursor++
		}
		return m, nil

	case tea.KeyEsc:
		m.rewindOverlay = nil
		return m, nil

	case tea.KeyEnter:
		item := m.rewindOverlay.Items[m.rewindOverlay.Cursor]
		m.rewindOverlay = nil

		// Truncate messages and history to before the selected message
		m.messages = m.messages[:item.MessageIndex]
		m.history = m.history[:item.HistoryIndex]

		// Populate textarea with the selected message text
		m.textarea.SetValue(item.FullText)

		m.saveConversation()
		m.refreshViewport()
		return m, nil
	}

	return m, nil
}

func handlePermissionKey(m *Model, msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.permission.Cursor > 0 {
			m.permission.Cursor--
			m.refreshViewport()
		}
		return m, nil

	case tea.KeyDown:
		if m.permission.Cursor < 2 {
			m.permission.Cursor++
			m.refreshViewport()
		}
		return m, nil

	case tea.KeyEnter:
		// Read cursor and tool call before clearing permission state
		cursor := m.permission.Cursor
		tc := m.awaitingPermission

		// Clear permission state
		m.permission = nil
		m.awaitingPermission = nil
		m.refreshViewport()

		switch cursor {
		case 0: // Allow
			return m, executeToolInterruptible(m.agent, *tc, m.interruptCh)

		case 1: // Always Allow
			m.alwaysAllow[tc.Function.Name] = true
			return m, executeToolInterruptible(m.agent, *tc, m.interruptCh)

		case 2: // Deny
			command := tc.Function.Name + ": " + tc.Function.Arguments
			result := "Tool call denied by user."

			// Append denial to history
			m.history = append(m.history, llm.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})

			// Append denied tool call to UI messages
			m.messages = append(m.messages, ChatEntry{
				Type:    EntryToolCall,
				Command: command,
				Denied:  true,
				Diff:    parseDiffFromToolCall(tc.Function.Name, tc.Function.Arguments, "", m.workingDir, true),
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
	}

	return m, nil
}
