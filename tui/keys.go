package tui

import (
	"fmt"
	"strings"
	"time"

	"go-tui/config"
	"go-tui/conversation"
	"go-tui/llm"
	"go-tui/tui/slashcmd"

	tea "github.com/charmbracelet/bubbletea"
)

func handleKeyMsg(m *Model, msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyShiftTab:
		m.bypassPermissions = !m.bypassPermissions
		m.refreshViewport()
		return m, nil
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

	// Model overlay mode
	if m.modelOverlay != nil {
		return handleModelOverlayKey(m, msg)
	}

	// Rewind overlay mode
	if m.rewindOverlay != nil {
		return handleRewindOverlayKey(m, msg)
	}

	// Conversation resume overlay mode
	if m.conversationOverlay != nil {
		return handleConversationOverlayKey(m, msg)
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

		// Send user input message to the model for processing
		return m, func() tea.Msg {
			return UserInputMsg{Text: text}
		}
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

		// Send rewind message to the model for processing
		return m, func() tea.Msg {
			return RewindToMessageMsg{
				MessageIndex: item.MessageIndex,
				HistoryIndex: item.HistoryIndex,
				FullText:     item.FullText,
			}
		}
	}

	return m, nil
}

func handleModelOverlayKey(m *Model, msg tea.KeyMsg) (*Model, tea.Cmd) {
	if m.modelOverlay.AwaitingKey {
		return handleModelKeyInput(m, msg)
	}

	switch msg.Type {
	case tea.KeyUp:
		if m.modelOverlay.Cursor > 0 {
			m.modelOverlay.Cursor--
		}
		return m, nil

	case tea.KeyDown:
		if m.modelOverlay.Cursor < len(m.modelOverlay.Items)-1 {
			m.modelOverlay.Cursor++
		}
		return m, nil

	case tea.KeyEsc:
		m.modelOverlay = nil
		return m, nil

	case tea.KeyEnter:
		if len(m.modelOverlay.Items) == 0 {
			m.modelOverlay = nil
			return m, nil
		}
		selected := m.modelOverlay.Items[m.modelOverlay.Cursor]

		// Check if we already have an API key for this model's provider
		md := config.ModelByName(selected)
		if md != nil {
			if key := m.settings.APIKey(md.APIKeyName); key != "" {
				m.modelOverlay = nil
				return m, activateModel(m, selected, key)
			}
		}

		// No key stored — prompt for it
		m.modelOverlay.AwaitingKey = true
		m.modelOverlay.SelectedModel = selected
		m.modelOverlay.KeyInput = ""
		return m, nil
	}

	return m, nil
}

func handleModelKeyInput(m *Model, msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Go back to model list
		m.modelOverlay.AwaitingKey = false
		m.modelOverlay.SelectedModel = ""
		m.modelOverlay.KeyInput = ""
		return m, nil

	case tea.KeyEnter:
		key := strings.TrimSpace(m.modelOverlay.KeyInput)
		if key == "" {
			return m, nil
		}
		selected := m.modelOverlay.SelectedModel
		m.modelOverlay = nil
		return m, activateModel(m, selected, key)

	case tea.KeyBackspace:
		if len(m.modelOverlay.KeyInput) > 0 {
			m.modelOverlay.KeyInput = m.modelOverlay.KeyInput[:len(m.modelOverlay.KeyInput)-1]
		}
		return m, nil

	case tea.KeyRunes:
		m.modelOverlay.KeyInput += string(msg.Runes)
		return m, nil
	}

	return m, nil
}

func activateModel(m *Model, name, apiKey string) tea.Cmd {
	return func() tea.Msg {
		md := config.ModelByName(name)
		if md == nil {
			return ModelSwitchedMsg{Err: fmt.Errorf("unknown model: %s", name)}
		}
		// Save API key under the provider key name (e.g. "Z_API")
		if err := m.settings.SetAPIKey(md.APIKeyName, apiKey); err != nil {
			return ModelSwitchedMsg{Err: fmt.Errorf("failed to save API key: %w", err)}
		}
		// Set active model
		if err := m.settings.SetActiveModel(name); err != nil {
			return ModelSwitchedMsg{Err: err}
		}
		llm.Configure(md.APIURL, apiKey, name)
		return ModelSwitchedMsg{Name: name}
	}
}

func handleConversationOverlayKey(m *Model, msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.conversationOverlay.Cursor > 0 {
			m.conversationOverlay.Cursor--
		}
		return m, nil

	case tea.KeyDown:
		if m.conversationOverlay.Cursor < len(m.conversationOverlay.Items)-1 {
			m.conversationOverlay.Cursor++
		}
		return m, nil

	case tea.KeyEsc:
		m.conversationOverlay = nil
		return m, nil

	case tea.KeyEnter:
		item := m.conversationOverlay.Items[m.conversationOverlay.Cursor]
		m.conversationOverlay = nil
		return m, func() tea.Msg {
			conv, err := conversation.Load(item.Path)
			if err != nil {
				return InterruptMsg{Reason: "Failed to load conversation: " + err.Error()}
			}
			return ResumeConversationMsg{Conv: conv}
		}
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

		// Send permission decision message to the model for processing
		decision := PermissionDecision(cursor)
		return m, func() tea.Msg {
			return PermissionDecisionMsg{
				Decision:    decision,
				ToolCall:    *tc,
				AlwaysAllow: decision == PermissionAlwaysAllow,
			}
		}
	}

	return m, nil
}
