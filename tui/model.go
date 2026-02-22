package tui

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"go-tui/agent"
	"go-tui/config"
	"go-tui/conversation"
	"go-tui/llm"
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
	FilePath     string `json:"file_path"`
	OldText      string `json:"old_text"`
	NewText      string `json:"new_text"`
	StartLine    int    `json:"start_line,omitempty"`
	BlockReplace bool   `json:"block_replace,omitempty"` // true for edit_file: show as block replacement
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
	alwaysAllow        map[string]bool
	toolRoundCount     int
	consecutiveErrors  int
	pendingToolCalls   []llm.ToolCall
	pendingToolIndex   int
	awaitingPermission *llm.ToolCall
	totalTokens        int
	streamingTokens    int
	streamingThinking  bool
	slashOverlay       *slashcmd.Overlay
	rewindOverlay      *slashcmd.RewindOverlay
	interruptCh        chan struct{}
	lastEscTime        time.Time
}

// separatorStyle and statusStyle are defined in theme.go

func New(workingDir string, conv *conversation.Data) Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.Focus()
	ta.ShowLineNumbers = false
	ta.SetHeight(config.TextareaHeight)
	ta.CharLimit = 0

	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = spinnerStyle

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

	return Model{
		textarea:         ta,
		spinner:          s,
		messages:         messages,
		agent:            a,
		conv:             conv,
		convDir:          conversation.Dir(workingDir),
		markdownRenderer: markdownRenderer,
		history:          history,
		workingDir:       workingDir,
		alwaysAllow:      make(map[string]bool),
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

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// statusLine(1) + separator(1) + textarea(3) + separator(1) = 6
		vpHeight := m.height - 6
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
		return handleKeyMsg(m, msg)

	case StreamTokenCountMsg:
		return m.handleStreamTokenCount(msg)

	case LLMResponseMsg:
		return m.handleLLMResponse(msg)

	case CompactResultMsg:
		return m.handleCompactResult(msg)

	case ToolResultMsg:
		return m.handleToolResult(msg)

	case InterruptMsg:
		return m.handleInterrupt(msg)

	case RewindToMessageMsg:
		return m.handleRewind(msg)

	case PermissionDecisionMsg:
		return m.handlePermissionDecision(msg)

	case UserInputMsg:
		return m.handleUserInput(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	var cmds []tea.Cmd
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

	if m.rewindOverlay != nil {
		vpView = m.rewindOverlay.View(m.width, m.viewport.Height)
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

	return lipgloss.JoinVertical(
		lipgloss.Left,
		vpView,
		statusLine,
		separator,
		inputArea,
		separator,
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

	// Right: <token label> <bar>
	tokenLabel := statusStyle.Render(fmt.Sprintf("%d/%d ", m.totalTokens, config.MaxContextTokens))
	barMaxWidth := m.width * 40 / 100
	if barMaxWidth < 1 {
		barMaxWidth = 1
	}
	displayTokens := m.totalTokens
	if displayTokens < 1000 {
		displayTokens = 1000
	}
	bar := renderBar(displayTokens, config.MaxContextTokens, barMaxWidth)
	right := tokenLabel + bar + "  "

	// Layout: <left> <gap> <right>
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := m.width - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func renderBar(value, max, width int) string {
	ratio := float64(value) / float64(max)
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	return lipgloss.NewStyle().Foreground(tokenBarColor(ratio)).Render(bar)
}
