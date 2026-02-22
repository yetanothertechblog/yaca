# YACA Codebase Analysis: Critical Gaps & Improvement Areas

A comprehensive analysis of the YACA (Yet Another Coding Assistant) codebase identifying areas for improvement.

---

## 🔴 CRITICAL ISSUES

### 1. No Test Coverage (80% of packages untested)

```
?   go-tui/agent/tools    [no test files]
?   go-tui/config         [no test files]
?   go-tui/llm            [no test files]
?   go-tui/lsp            [no test files]
?   go-tui/permissions    [no test files]
?   go-tui/settings       [no test files]
?   go-tui/tui/slashcmd   [no test files]
```

Only `agent`, `conversation`, and `tui` have tests. The LSP layer, tool execution, permissions, and settings have **zero tests**. This is unacceptable for a production coding assistant.

**Recommendation:**
- Add unit tests for all tool implementations in `agent/tools/`
- Add integration tests for LSP communication
- Add tests for permission matching logic
- Add tests for settings persistence

---

### 2. Hardcoded Single API Provider

```go
// config/models.go
const zaiEndpoint = "https://api.z.ai/api/paas/v4/chat/completions"
```

The entire codebase is hardcoded to a single proprietary API endpoint. No OpenAI, Anthropic, or local model support. This severely limits adoption.

**Recommendation:**
- Create a `Provider` interface with multiple implementations
- Support configuration-based provider registration
- Add OpenAI, Anthropic, and Ollama providers
- Allow custom API endpoints via configuration

---

### 3. No Context Window Management

```go
// tui/model.go:734
tokenLabel := statusStyle.Render(fmt.Sprintf("%d/%d Tokens", m.totalTokens, config.MaxContextTokens))
```

It **displays** token count but never actually:
- Truncates history when approaching limits
- Implements sliding window
- Warns the user before context overflow

The LLM will silently fail or produce degraded output when context exceeds limits.

**Recommendation:**
- Implement automatic history truncation when approaching token limits
- Add a warning system when context reaches 80% capacity
- Consider implementing a sliding window context strategy
- Add token estimation before sending requests

---

### 4. No Retry Logic or Resilience

```go
// llm/client.go:60-69
resp, err := http.DefaultClient.Do(httpReq)
if err != nil {
    return nil, fmt.Errorf("send request: %w", err)
}
```

No retries, no exponential backoff, no rate limiting, no circuit breaker. Network blips crash the entire request.

**Recommendation:**
- Implement exponential backoff with jitter
- Add configurable retry limits
- Implement rate limiting for API compliance
- Add circuit breaker pattern for fail-fast behavior
- Use context for request cancellation

---

### 5. Security: API Keys Stored in Plaintext

```go
// settings/settings.go:77
return os.WriteFile(filepath.Join(s.dir, settingsFile), data, 0o644)
```

API keys are stored in plaintext JSON at `~/.yaca/settings.json` with world-readable permissions (0644). No encryption, no keychain integration.

**Recommendation:**
- Integrate with OS keychain (Keychain on macOS, Credential Manager on Windows, Secret Service on Linux)
- At minimum, restrict file permissions to 0600
- Consider environment variable support for CI/CD scenarios
- Add key rotation support

---

## 🟠 MAJOR ISSUES

### 6. Monolithic TUI Model (746 lines)

`tui/model.go` is a god object handling:
- Message rendering
- Tool execution orchestration
- Permission management
- LLM streaming
- Slash commands
- Multiple overlay states
- Conversation persistence

**Recommendation:**
- Split into focused components:
  - `MessageRenderer` - handles message display
  - `ToolExecutor` - manages tool execution flow
  - `PermissionManager` - handles permission prompts
  - `StreamHandler` - manages LLM streaming
  - `OverlayManager` - manages UI overlays
- Use composition over inheritance
- Consider using the Elm architecture more strictly

---

### 7. No Streaming Cancellation Cleanup

```go
// tui/commands.go:179-181
case <-interruptCh:
    return InterruptMsg{Reason: "User interrupted"}
```

When user interrupts, the goroutine running `llm.CallLLMStream` continues running, consuming tokens and holding resources. No proper cancellation context.

**Recommendation:**
- Use `context.Context` for cancellation propagation
- Ensure HTTP request is cancelled when interrupted
- Clean up resources properly on interruption
- Consider implementing graceful shutdown for in-flight operations

---

### 8. Bash Tool: No Sandboxing

```go
// agent/tools/bash.go:40-41
cmd := exec.Command("bash", "-c", args.Command)
cmd.Dir = workingDir
```

No sandboxing, no resource limits (CPU/memory), no command allowlist beyond permissions. A malicious or confused LLM can run `rm -rf /` or fork-bomb the system.

**Recommendation:**
- Implement command allowlisting with patterns
- Add resource limits (CPU time, memory, process count)
- Consider using containers or namespaces for isolation
- Log all executed commands for audit trails
- Add timeout configuration per command type

---

### 9. LSP Diagnostics Timeout is Arbitrary

```go
// lsp/server.go:200-202
case <-time.After(5 * time.Second):
    // Timeout — return empty
```

5-second timeout may be too short for large files or slow language servers. No configuration, no retry.

**Recommendation:**
- Make timeout configurable
- Implement progressive timeout based on file size
- Add retry logic for transient failures
- Consider background diagnostics with notifications

---

### 10. No Diff Preview Before Applying Edits

The permission prompt shows the diff, but there's no way to:
- See full file context
- Edit the change manually
- Apply partially

**Recommendation:**
- Add option to open diff in external editor
- Implement partial diff application (hunk-level)
- Show surrounding context in diff preview
- Add "edit manually" option that opens the file

---

## 🟡 MODERATE ISSUES

### 11. No Conversation Branching

When rewinding, you lose the future messages. No way to:
- Branch from an earlier point
- Keep multiple conversation threads
- Compare different approaches

**Recommendation:**
- Implement conversation tree structure
- Add branch visualization in rewind overlay
- Allow naming branches for easy identification
- Support merging branches

---

### 12. Hardcoded Tool Limit

```go
// config/constants.go:6
MaxToolRounds = 100
```

Arbitrary limit with no configuration. Complex refactoring tasks can hit this.

**Recommendation:**
- Make limit configurable via settings
- Consider per-model limits based on context
- Add warning when approaching limit
- Allow user to continue if they choose

---

### 13. No Token Estimation

Token counting is done post-hoc via API response. No local estimation for:
- Predicting when context will overflow
- Cost estimation before sending
- Optimizing context packing

**Recommendation:**
- Implement tiktoken or similar for local token counting
- Add pre-send cost estimation
- Optimize context with intelligent truncation
- Warn users before expensive operations

---

### 14. Debug Logging to Working Directory

```go
// main.go:48-52
logDir := filepath.Join(workingDir, "log")
```

Creates a `log/` directory in every project. Should use standard OS locations like `~/.yaca/logs/` or system temp.

**Recommendation:**
- Use `~/.yaca/logs/` for all logs
- Or use OS-specific log directories (e.g., `/var/log/` on Linux)
- Add log rotation to prevent disk bloat
- Consider structured logging (JSON) for easier parsing

---

### 15. No Model Configuration Discovery

```go
// config/models.go:14-27
var SupportedModels = []ModelDef{...}
```

Models are hardcoded. No:
- Dynamic model discovery from API
- Custom model configuration file
- Ollama/local model support

**Recommendation:**
- Add configuration file for custom models
- Implement API-based model discovery
- Support Ollama and other local model providers
- Allow model-specific system prompts

---

### 16. Permission System is Coarse

```go
// permissions/permissions.go:84-104
func (p *Permissions) IsAllowed(toolName string, argsJSON string) bool {
```

Only supports:
- Full tool allowlisting
- Bash command prefix matching

No:
- Path-based permissions for file tools
- Argument inspection for read vs write
- Time-limited grants

**Recommendation:**
- Implement path-based permissions for file operations
- Add read/write distinction in permissions
- Support glob patterns for path matching
- Add session-based permissions (auto-expire)
- Implement permission scopes (full project, specific directories, etc.)

---

### 17. No Concurrent Tool Execution

Tools execute sequentially:
```go
// tui/model.go:510-512
m.pendingToolIndex++
cmd := m.dispatchNextTool()
```

Independent tools (e.g., reading multiple files) could run in parallel.

**Recommendation:**
- Analyze tool dependencies
- Execute independent tools concurrently
- Add concurrency limit (e.g., max 3 parallel tools)
- Show parallel execution in UI

---

### 18. Markdown Rendering is Synchronous

```go
// tui/model.go:291
m.viewport.SetContent(renderMessages(...))
```

Heavy markdown rendering blocks the UI. Should be offloaded.

**Recommendation:**
- Render markdown in background goroutine
- Implement progressive rendering
- Cache rendered content
- Consider WebGL/terminal-optimized rendering

---

## 🟢 MINOR ISSUES

### 19. No Keyboard Shortcut Customization

All keybindings hardcoded in `tui/keys.go`.

**Recommendation:**
- Add configuration file for keybindings
- Support keybinding profiles (vim, emacs, default)
- Allow runtime keybinding changes

---

### 20. No Mouse Support Beyond Scrolling

Can't click to select conversations, models, or permission options.

**Recommendation:**
- Add clickable UI elements
- Support mouse selection in overlays
- Add hover effects where applicable

---

### 21. Error Messages Expose Internals

```go
return ToolResult{}, NewToolError(ErrStringNotUnique, "old_string found multiple times",
    fmt.Sprintf("found %d times, must be unique. Include more surrounding context.", count))
```

Technical error codes exposed to end users.

**Recommendation:**
- Separate internal errors from user-facing messages
- Add error message localization support
- Provide actionable suggestions in error messages

---

### 22. No Accessibility Support

No screen reader support, no high-contrast mode, no font size adjustment.

**Recommendation:**
- Add screen reader announcements
- Implement high-contrast theme
- Make font size configurable
- Add keyboard navigation hints

---

### 23. Compact Uses Separate API Call

```go
// tui/commands.go:185-214
func compactHistory(history []llm.Message) tea.Cmd {
    messages := []llm.Message{...}
    result, err := llm.CallLLM(messages, nil)
```

Compact makes a separate API call with the full history, doubling token usage temporarily.

**Recommendation:**
- Use local summarization if possible
- Implement incremental compaction
- Consider hybrid approach (local outline + API details)

---

### 24. No Export/Import

Conversations stored in custom JSONL format. No export to:
- Markdown
- HTML
- PDF

**Recommendation:**
- Add `/export markdown` command
- Add `/export html` command
- Support conversation import
- Add conversation sharing (anonymized)

---

### 25. No Multi-File Edit Atomicity

When editing multiple files, a failure halfway through leaves the codebase in an inconsistent state. No rollback.

**Recommendation:**
- Implement transaction-like edit batching
- Add automatic rollback on failure
- Create backup before multi-file edits
- Add "undo last changes" command

---

## Summary Priority Matrix

| Priority | Category | Count |
|----------|----------|-------|
| 🔴 Critical | Testing, Security, Architecture | 5 |
| 🟠 Major | UX, Reliability, Safety | 5 |
| 🟡 Moderate | Features, Configuration | 7 |
| 🟢 Minor | Polish, Accessibility | 7 |

---

## Top 5 Immediate Actions

1. **Add tests** for `agent/tools`, `lsp`, `permissions`, `llm` packages
2. **Implement context window management** with automatic truncation
3. **Add multi-provider support** (OpenAI, Anthropic, Ollama)
4. **Secure API key storage** using OS keychain
5. **Add request retry** with exponential backoff

---

## Codebase Statistics

- **Total Lines of Go Code**: ~5,136
- **Packages**: 11
- **Test Coverage**: ~27% (3 of 11 packages tested)
- **Dependencies**: 20+ (primarily Charmbracelet ecosystem)

---

## Architecture Recommendations

### Immediate Refactoring
1. Split `tui/model.go` into smaller, focused components
2. Create provider abstraction for LLM clients
3. Add proper error types with user-friendly messages

### Medium-term Improvements
1. Implement plugin system for tools
2. Add configuration file support (YAML/TOML)
3. Create proper logging infrastructure

### Long-term Vision
1. Support multi-agent workflows
2. Add collaborative features
3. Implement persistent learning from corrections

---

*Generated from codebase analysis on 2024*
