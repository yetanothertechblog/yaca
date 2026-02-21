package tui

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-tui/agent"
	"go-tui/config"
	"go-tui/conversation"
	"go-tui/llm"
	"go-tui/permissions"
	"go-tui/settings"
	"go-tui/tui/slashcmd"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type EntryType int

const (
	EntryMessage EntryType = iota
	EntryToolCall
	EntryError
)

type DiffData struct {
	FilePath  string `json:"file_path"`
	OldText   string `json:"old_text"`
	NewText   string `json:"new_text"`
	StartLine int    `json:"start_line,omitempty"`
}

type ChatEntry struct {
	Type    EntryType `json:"type"`
	Role    string    `json:"role,omitempty"`
	Content string    `json:"content,omitempty"`
	Command string    `json:"command,omitempty"`
	Result  string    `json:"result,omitempty"`
	Denied  bool      `json:"denied,omitempty"`
	Diff    *DiffData `json:"diff,omitempty"`
	Error   string    `json:"error,omitempty"`
}

const maxToolRounds = config.MaxToolRounds
const maxConsecutiveErrors = 3

type Model struct {
	viewport           viewport.Model
	textarea           textarea.Model
	spinner            spinner.Model
	messages           []ChatEntry
	agent              *agent.Agent
	waiting            bool
	width              int
	height             int
	ready              bool
	permission         *PermissionPrompt
	conv               *conversation.Data
	convDir            string
	markdownRenderer   *MarkdownRenderer
	history            []llm.Message
	workingDir         string
	permissions        *permissions.Permissions
	toolRoundCount     int
	consecutiveErrors  int
	pendingToolCalls   []llm.ToolCall
	pendingToolIndex   int
	awaitingPermission *llm.ToolCall
	totalTokens        int
	streamingTokens    int
	streamingThinking  bool
	slashOverlay         *slashcmd.Overlay
	rewindOverlay        *slashcmd.RewindOverlay
	conversationOverlay  *slashcmd.RewindOverlay
	modelOverlay         *slashcmd.ModelOverlay
	settings           *settings.Settings
	interruptCh        chan struct{}
	lastEscTime        time.Time
	bypassPermissions  bool
}

// separatorStyle and statusStyle are defined in theme.go

func New(workingDir string, conv *conversation.Data, s *settings.Settings) Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.Focus()
	ta.ShowLineNumbers = false
	ta.SetHeight(config.TextareaHeight)
	ta.CharLimit = 0

	sp := spinner.New()
	sp.Spinner = spinner.Points
	sp.Style = spinnerStyle

	// Initialize markdown renderer before the TUI event loop starts,
	// so the terminal color query (from "auto" style) completes before
	// the textarea captures input.
	mdStart := time.Now()
	markdownRenderer, err := NewMarkdownRenderer(0)
	log.Printf("NewMarkdownRenderer took %s", time.Since(mdStart))
	if err != nil {
		log.Printf("failed to initialize markdown renderer: %v", err)
		markdownRenderer = nil
	}

	var messages []ChatEntry
	if err := json.Unmarshal(conv.UIMessages, &messages); err != nil {
		log.Printf("failed to unmarshal UI messages: %v", err)
		messages = []ChatEntry{}
	}

	a := agent.New(workingDir)

	var history []llm.Message
	if err := json.Unmarshal(conv.AgentHistory, &history); err != nil {
		log.Printf("failed to unmarshal agent history: %v", err)
	}

	perms, err := permissions.Load(workingDir)
	if err != nil {
		log.Printf("failed to load permissions: %v", err)
	}

	return Model{
		textarea:         ta,
		spinner:          sp,
		messages:         messages,
		agent:            a,
		conv:             conv,
		convDir:          conversation.Dir(),
		markdownRenderer: markdownRenderer,
		history:          history,
		workingDir:       workingDir,
		permissions:      perms,
		settings:         s,
	}
}

func (m *Model) Shutdown() {
	m.agent.Shutdown()
}

func (m *Model) saveConversation() {
	uiJSON, err := json.Marshal(m.messages)
	if err != nil {
		log.Printf("failed to marshal UI messages: %v", err)
		return
	}
	histJSON, err := json.Marshal(m.history)
	if err != nil {
		log.Printf("failed to marshal agent history: %v", err)
		return
	}
	m.conv.UIMessages = uiJSON
	m.conv.AgentHistory = histJSON
	if err := m.conv.Save(m.convDir); err != nil {
		log.Printf("failed to save conversation: %v", err)
	}
}

// appendError attaches an error message to the last chat entry as an indented block,
// or creates a standalone entry if there are no previous messages.
func (m *Model) appendError(errMsg string) {
	if len(m.messages) > 0 {
		m.messages[len(m.messages)-1].Error = errMsg
	} else {
		m.messages = append(m.messages, ChatEntry{
			Type:    EntryMessage,
			Role:    "assistant",
			Content: "",
			Error:   errMsg,
		})
	}
}

func parseDiffFromToolCall(toolName, args, result, workingDir string, denied bool) *DiffData {
	if denied {
		return parseDiffFromArgs(toolName, args, workingDir)
	}

	if result == "" {
		return nil
	}

	switch toolName {
	case "write_file":
		var r struct {
			FilePath   string `json:"file_path"`
			NewContent string `json:"new_content"`
		}
		if json.Unmarshal([]byte(result), &r) != nil || r.FilePath == "" {
			return parseDiffFromArgs(toolName, args, workingDir)
		}
		return &DiffData{
			FilePath: r.FilePath,
			NewText:  r.NewContent,
		}
	case "edit_file":
		var r struct {
			FilePath  string `json:"file_path"`
			OldString string `json:"old_string"`
			NewString string `json:"new_string"`
		}
		if json.Unmarshal([]byte(result), &r) != nil || r.FilePath == "" {
			return parseDiffFromArgs(toolName, args, workingDir)
		}
		startLine := 1
		path := r.FilePath
		if !filepath.IsAbs(path) {
			path = filepath.Join(workingDir, path)
		}
		if data, err := os.ReadFile(path); err == nil {
			startLine = findStartLine(string(data), r.OldString)
		}
		return &DiffData{
			FilePath:  r.FilePath,
			OldText:   r.OldString,
			NewText:   r.NewString,
			StartLine: startLine,
		}
	}
	return nil
}

func parseDiffFromArgs(name, argsJSON, workingDir string) *DiffData {
	switch name {
	case "edit_file":
		var args struct {
			FilePath  string `json:"file_path"`
			OldString string `json:"old_string"`
			NewString string `json:"new_string"`
		}
		if json.Unmarshal([]byte(argsJSON), &args) != nil || args.FilePath == "" {
			return nil
		}
		startLine := 1
		path := args.FilePath
		if !filepath.IsAbs(path) {
			path = filepath.Join(workingDir, path)
		}
		if data, err := os.ReadFile(path); err == nil {
			startLine = findStartLine(string(data), args.OldString)
		}
		return &DiffData{
			FilePath:  args.FilePath,
			OldText:   args.OldString,
			NewText:   args.NewString,
			StartLine: startLine,
		}

	case "write_file":
		var args struct {
			FilePath string `json:"file_path"`
			Content  string `json:"content"`
		}
		if json.Unmarshal([]byte(argsJSON), &args) != nil || args.FilePath == "" {
			return nil
		}
		return &DiffData{
			FilePath: args.FilePath,
			NewText:  args.Content,
		}
	}
	return nil
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, spinner.Tick)
}

func (m *Model) updateMarkdownRenderer() {
	if r, err := NewMarkdownRenderer(m.width); err == nil {
		m.markdownRenderer = r
	}
}

func (m *Model) refreshViewport() {
	m.viewport.SetContent(renderMessages(m.messages, m.permission, m.width, m.markdownRenderer))
	m.viewport.GotoBottom()
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

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// statusLine(1) + separator(1) + textarea(3) + separator(1) + bypassLine(1) = 7
		vpHeight := m.height - 7
		taWidth := m.width

		if !m.ready {
			m.viewport = viewport.New(m.width, vpHeight)
			m.textarea.SetWidth(taWidth)
			m.updateMarkdownRenderer()
			m.refreshViewport()
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = vpHeight
			m.textarea.SetWidth(taWidth)
			m.updateMarkdownRenderer()
			m.refreshViewport()
		}

		return m, nil

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		var cmd tea.Cmd
		m, cmd = handleKeyMsg(m, msg)
		return m, cmd

	case StreamTokenCountMsg:
		m.streamingTokens = msg.Count
		m.streamingThinking = msg.Thinking
		return m, waitForStreamInterruptible(msg.ch, m.interruptCh)

	case LLMResponseMsg:
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
			if strings.TrimSpace(msg.Content) != "" {
				m.messages = append(m.messages, ChatEntry{
					Type:    EntryMessage,
					Role:    "assistant",
					Content: msg.Content,
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
		if strings.TrimSpace(msg.Content) != "" {
			m.messages = append(m.messages, ChatEntry{
				Type:    EntryMessage,
				Role:    "assistant",
				Content: msg.Content,
			})
		}

		m.pendingToolCalls = msg.ToolCalls
		m.pendingToolIndex = 0
		m.toolRoundCount++
		log.Printf("LLM round %d: %d tool calls, content=%q", m.toolRoundCount, len(msg.ToolCalls), msg.Content)
		m.refreshViewport()

		cmd := m.dispatchNextTool()
		return m, cmd

	case CompactResultMsg:
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

	case ToolResultMsg:
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

	case InterruptMsg:
		m.waiting = false
		m.textarea.Focus()
		m.appendError(msg.Reason)
		m.saveConversation()
		m.refreshViewport()
		return m, nil

	case ModelSwitchedMsg:
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

	case RewindToMessageMsg:
		// Truncate messages and history to before the selected message
		m.messages = m.messages[:msg.MessageIndex]
		m.history = m.history[:msg.HistoryIndex]

		// Populate textarea with the selected message text
		m.textarea.SetValue(msg.FullText)

		m.saveConversation()
		m.refreshViewport()
		return m, nil

	case PermissionDecisionMsg:
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

	case ResumeConversationMsg:
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

	case UserInputMsg:
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

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	statusLine := m.renderStatusLine()

	separator := separatorStyle.Render(strings.Repeat("─", m.width))

	inputArea := m.textarea.View()

	vpView := m.viewport.View()

	if m.modelOverlay != nil {
		vpView = m.modelOverlay.View(m.width, m.viewport.Height)
	} else if m.rewindOverlay != nil {
		vpView = m.rewindOverlay.View(m.width, m.viewport.Height)
	} else if m.conversationOverlay != nil {
		vpView = m.conversationOverlay.View(m.width, m.viewport.Height)
	} else if m.slashOverlay != nil {
		overlay := m.slashOverlay.View(m.width)
		if overlay != "" {
			overlayLines := strings.Split(overlay, "\n")
			vpLines := strings.Split(vpView, "\n")
			overlayH := len(overlayLines)
			if overlayH > 0 && len(vpLines) >= overlayH {
				copy(vpLines[len(vpLines)-overlayH:], overlayLines)
			}
			vpView = strings.Join(vpLines, "\n")
		}
	}

	bypassLine := " "
	if m.bypassPermissions {
		bypassLine = bypassStyle.Render("⚡ BYPASS MODE (Shift+Tab to disable)")
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		vpView,
		statusLine,
		separator,
		inputArea,
		separator,
		bypassLine,
	)
}

func (m *Model) renderStatusLine() string {
	// Left: thinking/streaming indicator
	var left string
	if m.waiting {
		if m.streamingTokens > 0 {
			thinkingStr := ""
			if m.streamingThinking {
				thinkingStr = " · ( thinking )"
			}
			left = m.spinner.View() + fmt.Sprintf(" Processing · ⬇ %d tokens%s (Double ESC to interrupt)", m.streamingTokens, thinkingStr)
		} else {
			left = m.spinner.View() + " Processing (Double ESC to interrupt)"
		}
	}

	// Right: <model name> · <token counter>
	modelLabel := ""
	if name := m.settings.ActiveModel(); name != "" {
		modelLabel = statusStyle.Render(name + " · ")
	}
	tokenLabel := statusStyle.Render(fmt.Sprintf("%d/%d Tokens", m.totalTokens, config.MaxContextTokens))
	right := modelLabel + tokenLabel + "  "

	// Layout: <left> <gap> <right>
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := m.width - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}
