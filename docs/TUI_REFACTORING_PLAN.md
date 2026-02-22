# TUI Model Refactoring Plan

## Overview

The `tui/model.go` file is **685 lines** with a ~275-line `Update` method that handles all message types in a single type switch. The file mixes application state transitions (LLM responses, tool dispatch, permissions) with UI concerns (viewport, textarea, spinner). This plan splits `Update` into per-message handler methods in separate files — no new types, no new abstractions.

## Current State

### File Layout

| File | Lines | Responsibility |
|------|-------|----------------|
| `model.go` | 685 | Model struct, `New()`, `Update()`, `View()`, `dispatchNextTool()`, `saveConversation()`, `refreshViewport()`, `renderStatusLine()`, diff parsing |
| `keys.go` | 248 | `handleKeyMsg()`, overlay key handlers, permission key handler |
| `messages.go` | 253 | `renderMessages()`, `renderToolCallEntry()`, `renderMessageEntry()`, `formatCommand()` |
| `commands.go` | 232 | `callLLMInterruptible()`, `executeToolInterruptible()`, `compactHistory()`, stream/interrupt plumbing |
| `slashexec.go` | 88 | `executeSlashCommand()`, `executeRewind()` |
| `permission.go` | 58 | `PermissionPrompt` struct and `View()` |
| `diff.go` | 133 | Diff rendering |
| `grouping.go` | 72 | Tool call grouping logic |
| `theme.go` | 105 | Styles |
| `markdown.go` | 51 | Markdown renderer wrapper |

### Current Model Struct

```go
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
```

### The Problem

The `Update` method is a ~275-line type switch handling 10+ message types:
- `tea.WindowSizeMsg` — viewport/textarea resizing
- `tea.MouseMsg` — viewport scroll
- `tea.KeyMsg` — delegated to `handleKeyMsg`
- `StreamTokenCountMsg` — streaming state update
- `LLMResponseMsg` — LLM response handling, tool call dispatch
- `CompactResultMsg` — conversation compaction
- `ToolResultMsg` — tool result processing, next tool dispatch
- `InterruptMsg` — interrupt cleanup
- `RewindToMessageMsg` — conversation rewind
- `PermissionDecisionMsg` — permission allow/deny/always-allow
- `UserInputMsg` — user message submission
- `spinner.TickMsg` — spinner animation

Each case block is 10-40 lines of state mutation + side effects. Reading `Update` requires understanding all message types at once, even when working on just one.

---

## Approach: Split Update Into Handler Methods

Move each `case` block into a dedicated method on Model, grouped into files by concern. No new types. No new abstractions. Just methods on the same struct, in separate files.

### Principles

1. **No new types** — Model stays as the single Bubbletea model
2. **Methods, not objects** — each handler is `func (m *Model) handleFoo(msg FooMsg) (tea.Model, tea.Cmd)`
3. **File per concern** — group related handlers into files named by what they handle
4. **Preserve behavior** — pure mechanical extraction, zero logic changes
5. **Already-factored code stays put** — `keys.go`, `messages.go`, `commands.go`, `slashexec.go` are fine as-is

---

## Phase 1: Extract LLM/Stream Handlers

**Risk**: Low (mechanical move)

### Create `tui/handle_llm.go`

Move the following `case` blocks out of `Update` into methods:

```go
// tui/handle_llm.go
package tui

func (m *Model) handleStreamTokenCount(msg StreamTokenCountMsg) (tea.Model, tea.Cmd) {
    m.streamingTokens = msg.Count
    m.streamingThinking = msg.Thinking
    return m, waitForStreamInterruptible(msg.ch, m.interruptCh)
}

func (m *Model) handleLLMResponse(msg LLMResponseMsg) (tea.Model, tea.Cmd) {
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

    // Has tool calls
    m.history = append(m.history, llm.Message{
        Role:      "assistant",
        Content:   msg.Content,
        ToolCalls: msg.ToolCalls,
    })
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
    m.refreshViewport()
    return m, m.dispatchNextTool()
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
        {Role: "user", Content: "[Conversation summary]\n" + msg.Summary},
    }
    m.messages = []ChatEntry{
        {Type: EntryMessage, Role: "assistant", Content: "Conversation compacted:\n\n" + msg.Summary},
    }
    if msg.Usage != nil {
        m.totalTokens = msg.Usage.TotalTokens
    }
    m.saveConversation()
    m.refreshViewport()
    return m, nil
}
```

### Update `model.go`

The `Update` method becomes:

```go
case StreamTokenCountMsg:
    return m.handleStreamTokenCount(msg)
case LLMResponseMsg:
    return m.handleLLMResponse(msg)
case CompactResultMsg:
    return m.handleCompactResult(msg)
```

**Lines moved out of `model.go`**: ~100

---

## Phase 2: Extract Tool/Permission Handlers

**Risk**: Low (mechanical move)

### Create `tui/handle_tools.go`

```go
// tui/handle_tools.go
package tui

func (m *Model) handleToolResult(msg ToolResultMsg) (tea.Model, tea.Cmd) {
    // Current ToolResultMsg case block (~40 lines)
    // ...
}

func (m *Model) handlePermissionDecision(msg PermissionDecisionMsg) (tea.Model, tea.Cmd) {
    // Current PermissionDecisionMsg case block (~35 lines)
    // ...
}
```

Also move `dispatchNextTool` into this file — it's tool dispatch logic, not general model logic.

### Update `model.go`

```go
case ToolResultMsg:
    return m.handleToolResult(msg)
case PermissionDecisionMsg:
    return m.handlePermissionDecision(msg)
```

**Lines moved out of `model.go`**: ~110 (including `dispatchNextTool`)

---

## Phase 3: Extract User Input / Lifecycle Handlers

**Risk**: Low (mechanical move)

### Create `tui/handle_input.go`

```go
// tui/handle_input.go
package tui

func (m *Model) handleUserInput(msg UserInputMsg) (tea.Model, tea.Cmd) {
    // Current UserInputMsg case block (~15 lines)
    // ...
}

func (m *Model) handleInterrupt(msg InterruptMsg) (tea.Model, tea.Cmd) {
    // Current InterruptMsg case block (~6 lines)
    // ...
}

func (m *Model) handleRewind(msg RewindToMessageMsg) (tea.Model, tea.Cmd) {
    // Current RewindToMessageMsg case block (~8 lines)
    // ...
}
```

**Lines moved out of `model.go`**: ~40

---

## Phase 4: Extract Diff Parsing From model.go

**Risk**: Low

`parseDiffFromToolCall` and `parseDiffFromArgs` (~95 lines) are pure functions that don't touch Model state. They're currently in `model.go` but belong in `diff.go` alongside `renderDiff`.

### Move to `tui/diff.go`

Move `parseDiffFromToolCall()` and `parseDiffFromArgs()` to `diff.go`. Also move `findStartLine()` if it's only used by diff parsing.

**Lines moved out of `model.go`**: ~95

---

## Result

### Before

```
model.go: 685 lines
  - Update():     ~275 lines (10+ case blocks)
  - Diff parsing: ~95 lines
  - View/render:  ~85 lines
  - Init/New:     ~50 lines
  - Helpers:      ~30 lines
  - Struct/types: ~50 lines
```

### After

```
model.go:         ~345 lines (struct, New, Init, Update dispatch, View, renderStatusLine, saveConversation, helpers)
handle_llm.go:    ~100 lines (handleStreamTokenCount, handleLLMResponse, handleCompactResult)
handle_tools.go:  ~110 lines (handleToolResult, handlePermissionDecision, dispatchNextTool)
handle_input.go:  ~40 lines  (handleUserInput, handleInterrupt, handleRewind)
diff.go:          ~228 lines (existing renderDiff + moved parseDiffFromToolCall, parseDiffFromArgs)
```

The `Update` method shrinks from ~275 lines to ~40 lines:

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        // 15 lines — stays inline, it's UI setup
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
```

---

## Testing Strategy

Each handler method can be tested by constructing a Model with the relevant fields set, calling the handler, and asserting the resulting state. No mocking needed — these are methods on a concrete struct.

Example:

```go
func TestHandleInterrupt(t *testing.T) {
    m := &Model{waiting: true, messages: []ChatEntry{}}
    result, cmd := m.handleInterrupt(InterruptMsg{Reason: "User interrupted"})
    model := result.(*Model)
    if model.waiting {
        t.Error("expected waiting to be false")
    }
    if cmd != nil {
        t.Error("expected no command")
    }
}
```

Focus testing on:
- `handleLLMResponse` — the most complex handler (tool call branching)
- `handleToolResult` — error accumulation, next-tool dispatch
- `handlePermissionDecision` — allow/deny/always-allow paths
- `dispatchNextTool` — round limit, permission gating

---

## Migration Checklist

- [ ] Phase 1: Create `handle_llm.go`, move 3 handlers, update `Update()`
- [ ] Phase 2: Create `handle_tools.go`, move 2 handlers + `dispatchNextTool`
- [ ] Phase 3: Create `handle_input.go`, move 3 handlers
- [ ] Phase 4: Move diff parsing functions to `diff.go`
- [ ] Verify: `go build ./...` passes after each phase
- [ ] Verify: `go test ./...` passes after each phase
- [ ] Verify: no functional changes (diff of before/after behavior should be empty)

---

## What This Plan Does NOT Do

These are explicitly out of scope:

- **No new types or abstractions** — no `ToolExecutor`, `StreamHandler`, `OverlayManager`, etc.
- **No interface extraction** — the Model struct stays concrete
- **No logic changes** — pure file reorganization
- **No rendering changes** — `messages.go` rendering is already well-factored
- **No `commands.go` changes** — the goroutine/channel plumbing is already in its own file
- **No `keys.go` changes** — key handling is already extracted

---

## Related Issues

- IMPROVEMENTS.md item #6: Monolithic TUI Model

---

*Last Updated: 2026-02-22*
