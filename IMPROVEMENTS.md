# YACA Codebase Analysis: Critical Gaps & Improvement Areas

A comprehensive analysis of the YACA (Yet Another Coding Assistant) codebase identifying areas for improvement.

---

## 🔴 CRITICAL ISSUES

### 1. ✅ No Test Coverage (COMPLETED)

**Status:** Comprehensive test coverage added

**Implementation:**
- Added 8 test files for `agent/tools` package covering all tool implementations
  - `bash_test.go` - Bash command execution (8 tests)
  - `readfile_test.go` - File reading with pagination (10 tests)
  - `writefile_test.go` - File creation/overwriting (7 tests)
  - `editfile_test.go` - String replacement editing (8 tests)
  - `listfiles_test.go` - Directory listing (9 tests)
  - `search_test.go` - Grep-based search (8 tests)
  - `registry_test.go` - Tool registration system
  - `errors_test.go` - ToolError types
  - `tool_test.go` - Core tool infrastructure
  - **Coverage: 88.3%**

- Added tests for `llm` package
  - `types_test.go` - LLM message/request/response types (4 tests)
  - Tests Message, ChatRequest, Usage, Tool, Delta serialization

- Added tests for `lsp` package
  - `types_test.go` - LSP type structures (5 tests)
  - `jsonrpc_test.go` - JSON-RPC 2.0 protocol (5 tests)
  - Tests Diagnostic, Range, Position, Request, Response, Notification

**Still untested:**
- `config` - configuration constants
- `permissions` - permission matching logic
- `settings` - settings persistence
- `tui/slashcmd` - slash command handling

---

### 2. ✅ Hardcoded Single API Provider (COMPLETED)

**Status:** Multi-provider support implemented

**Implementation:**
- Removed hardcoded `zaiEndpoint` constant
- Each model now has its own `APIURL` field
- Added OpenAI-compatible models: gpt-4o, gpt-4o-mini, gpt-4-turbo, gpt-3.5-turbo, codex-5.3
- Supports different API keys per provider (Z_API, OPENAI_API)
- Maintains backward compatibility with existing Z.AI models

~~The entire codebase is hardcoded to a single proprietary API endpoint. No OpenAI, Anthropic, or local model support. This severely limits adoption.~~

~~**Recommendation:~~
~~- Create a `Provider` interface with multiple implementations~~
~~- Support configuration-based provider registration~~
~~- Add OpenAI, Anthropic, and Ollama providers~~
~~- Allow custom API endpoints via configuration~~

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

### 4. ✅ No Retry Logic or Resilience (COMPLETED)

**Status:** Implemented with exponential backoff and circuit breaker

**Implementation:**
- Created `retry/` package with configurable exponential backoff and jitter
- Created `circuit/` package with full circuit breaker pattern (CLOSED, OPEN, HALF_OPEN states)
- Added configuration constants in `config/constants.go` for retry and circuit breaker settings
- Integrated both patterns into `llm/client.go` for `CallLLM` and `CallLLMStream`
- Smart retry detection: only retries on network/transient errors (connection refused, timeout, rate limits, etc.)
- Automatic circuit opening after consecutive failures with configurable threshold
- Automatic recovery through half-open state that probes service health
- Added helper functions `GetCircuitBreakerStats()` and `ResetCircuitBreaker()` for monitoring and manual control
- Comprehensive test coverage for both packages

~~```go
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
- Use context for request cancellation~~

---

### 5. ✅ Security: API Keys Stored in Plaintext (COMPLETED)

**Status:** Fixed in PR #10 - [secure-api-key-storage](https://github.com/yetanothertechblog/go-tui-agent/pull/10)

**Implementation:**
- Created `settings/keystore.go` with OS keychain integration using `go-keyring`
- Supports macOS Keychain, Windows Credential Manager, Linux Secret Service
- Priority chain: Environment variables (`YACA_<KEYNAME>_API_KEY`) → OS keychain → In-memory cache
- Restricted file permissions from `0644` to `0600`
- Added `Get()`, `Set()`, `Delete()`, `IsKeyringAvailable()` methods

~~API keys are stored in plaintext JSON at `~/.yaca/settings.json` with world-readable permissions (0644). No encryption, no keychain integration.~~

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

---

### 11. No Token Estimation

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

### 12. Markdown Rendering is Synchronous

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

### 13. No Keyboard Shortcut Customization

All keybindings hardcoded in `tui/keys.go`.

**Recommendation:**
- Add configuration file for keybindings
- Support keybinding profiles (vim, emacs, default)
- Allow runtime keybinding changes

---

### 14. Error Messages Expose Internals

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

### 15. Compact Uses Separate API Call

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

### 16. No Export/Import

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

## Summary Priority Matrix

| Priority | Category | Count |
|----------|----------|-------|
| 🔴 Critical | Testing, Security, Architecture | 3 |
| 🟠 Major | UX, Reliability, Safety | 5 |
| 🟡 Moderate | Features, Configuration | 2 |
| 🟢 Minor | Polish, Accessibility | 4 |

---

## Top 5 Immediate Actions

1. ✅ **Add tests** for `agent/tools`, `lsp`, `llm` packages (PR improving_tests)
2. **Implement context window management** with automatic truncation
3. ✅ **Add multi-provider support** (OpenAI compatible models)
4. ✅ **Secure API key storage** using OS keychain (PR #10)
5. ✅ **Add request retry** with exponential backoff and circuit breaker

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
