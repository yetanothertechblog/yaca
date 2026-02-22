package tui

import (
	"strings"
	"testing"

	"go-tui/llm"
	"go-tui/permissions"
	"go-tui/settings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// newBypassTestModel returns a minimal Model suitable for bypass tests.
func newBypassTestModel(t *testing.T) *Model {
	t.Helper()
	perms, err := permissions.Load(t.TempDir())
	if err != nil {
		t.Fatalf("permissions.Load: %v", err)
	}
	s, err := settings.Load()
	if err != nil {
		t.Fatalf("settings.Load: %v", err)
	}
	ta := textarea.New()
	ta.SetWidth(80)
	ta.SetHeight(3)
	sp := spinner.New()
	return &Model{
		permissions: perms,
		settings:    s,
		width:       80,
		height:      24,
		ready:       true,
		viewport:    viewport.New(80, 17), // 24 - 7
		textarea:    ta,
		spinner:     sp,
	}
}

// TestBypassKeyToggle verifies that Shift+Tab flips bypassPermissions each press.
func TestBypassKeyToggle(t *testing.T) {
	m := newBypassTestModel(t)

	if m.bypassPermissions {
		t.Fatal("bypassPermissions should start false")
	}

	shiftTab := tea.KeyMsg{Type: tea.KeyShiftTab}

	m, _ = handleKeyMsg(m, shiftTab)
	if !m.bypassPermissions {
		t.Error("bypassPermissions should be true after first Shift+Tab")
	}

	m, _ = handleKeyMsg(m, shiftTab)
	if m.bypassPermissions {
		t.Error("bypassPermissions should be false after second Shift+Tab")
	}
}

// TestBypassSkipsPermissionPrompt verifies that when bypass is active,
// dispatchNextTool returns a command and does not show a permission prompt.
func TestBypassSkipsPermissionPrompt(t *testing.T) {
	m := newBypassTestModel(t)
	m.bypassPermissions = true
	m.interruptCh = make(chan struct{})
	// agent is nil; executeToolInterruptible returns a lazy cmd so it won't panic here.
	m.pendingToolCalls = []llm.ToolCall{
		{ID: "tc1", Function: llm.ToolCallFunction{Name: "bash", Arguments: `{"command":"echo hello"}`}},
	}

	cmd := m.dispatchNextTool()

	if cmd == nil {
		t.Error("expected a non-nil command (tool execution) when bypass is active")
	}
	if m.permission != nil {
		t.Error("expected no permission prompt when bypass is active")
	}
}

// TestNoBypassShowsPermissionPrompt verifies that without bypass mode an
// unallowed tool shows the permission prompt and returns no command.
func TestNoBypassShowsPermissionPrompt(t *testing.T) {
	m := newBypassTestModel(t)
	m.bypassPermissions = false
	m.interruptCh = make(chan struct{})
	m.pendingToolCalls = []llm.ToolCall{
		{ID: "tc1", Function: llm.ToolCallFunction{Name: "bash", Arguments: `{"command":"echo hello"}`}},
	}

	cmd := m.dispatchNextTool()

	if cmd != nil {
		t.Error("expected nil command when permission prompt is shown")
	}
	if m.permission == nil {
		t.Error("expected permission prompt to be set when bypass is inactive")
	}
}

// TestBypassViewIndicator checks that the bypass line appears in the rendered
// view exactly when bypass mode is active.
func TestBypassViewIndicator(t *testing.T) {
	m := newBypassTestModel(t)

	view := stripANSI(m.View())
	if strings.Contains(view, "Bypass Permissions") {
		t.Error("bypass indicator should not appear when bypass is off")
	}

	m.bypassPermissions = true
	view = stripANSI(m.View())
	if !strings.Contains(view, "Bypass Permissions") {
		t.Error("bypass indicator should appear when bypass is on")
	}
}
