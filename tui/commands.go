package tui

import (
	"log"
	"strings"
	"sync"
	"time"

	"go-tui/agent"
	"go-tui/agent/tools"
	"go-tui/conversation"
	"go-tui/llm"

	tea "github.com/charmbracelet/bubbletea"
)

// Messages

type LLMResponseMsg struct {
	Content          string
	ReasoningContent string
	ToolCalls        []llm.ToolCall
	Usage            *llm.Usage
	Err              error
}

type ToolResultMsg struct {
	ToolCallID string
	ToolName   string
	Args       string
	Result     string
	Err        error
}

// StreamChunkMsg carries debounced content from the LLM stream.
// The producer goroutine batches fragments with a 16ms AfterFunc timer so the
// model only receives a message after the stream has been quiet for 16ms.
// Regular assistant tokens go in Content; extended-thinking tokens go in ThinkingContent.
type StreamChunkMsg struct {
	Content         string
	ThinkingContent string
	Thinking        bool
	ch              <-chan tea.Msg // continuation channel
}

type CompactResultMsg struct {
	Summary string
	Usage   *llm.Usage
	Err     error
}

type InterruptMsg struct {
	Reason string
}

type PermissionDecision int

type RewindToMessageMsg struct {
	MessageIndex int
	HistoryIndex int
	FullText     string
}

type PermissionDecisionMsg struct {
	Decision    PermissionDecision
	ToolCall    llm.ToolCall
	AlwaysAllow bool
}

type UserInputMsg struct {
	Text string
}

type ModelSwitchedMsg struct {
	Name string
	Err  error
}

type ResumeConversationMsg struct {
	Conv *conversation.Data
}

const (
	PermissionAllow PermissionDecision = iota
	PermissionAlwaysAllow
	PermissionDeny
)

// Cmd factories

func callLLMInterruptible(a *agent.Agent, history []llm.Message, interruptCh <-chan struct{}) tea.Cmd {
	messages := make([]llm.Message, 0, len(history)+1)
	messages = append(messages, llm.Message{
		Role:    "system",
		Content: a.SystemPrompt(),
	})
	messages = append(messages, history...)

	ch := make(chan tea.Msg, 1000)

	go func() {
		defer close(ch)

		var mu sync.Mutex
		var buf strings.Builder
		var thinkingBuf strings.Builder
		var thinking bool
		var timer *time.Timer

		// send drains both buffers and emits a StreamChunkMsg.
		// Must NOT be called with mu held.
		send := func() {
			mu.Lock()
			chunk := buf.String()
			thinkingChunk := thinkingBuf.String()
			isThinking := thinking
			buf.Reset()
			thinkingBuf.Reset()
			mu.Unlock()
			if chunk == "" && thinkingChunk == "" {
				return
			}
			select {
			case ch <- StreamChunkMsg{Content: chunk, ThinkingContent: thinkingChunk, Thinking: isThinking, ch: ch}:
			default: // drop if full; LLMResponseMsg carries the complete text
			}
		}

		onContent := func(content string, isThinking bool) {
			mu.Lock()
			if isThinking {
				thinkingBuf.WriteString(content)
			} else {
				buf.WriteString(content)
			}
			thinking = isThinking
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(16*time.Millisecond, send)
			mu.Unlock()
		}

		result, err := llm.CallLLMStream(messages, tools.All(), onContent)

		// Cancel the pending timer and flush any remaining buffered content.
		mu.Lock()
		if timer != nil {
			timer.Stop()
			timer = nil
		}
		mu.Unlock()
		send()

		if err != nil {
			log.Printf("llm error: %v", err)
			ch <- LLMResponseMsg{Err: err}
			return
		}
		ch <- LLMResponseMsg{
			Content:          result.Delta.Content,
			ReasoningContent: result.Delta.ReasoningContent,
			ToolCalls:        result.Delta.ToolCalls,
			Usage:            result.Usage,
		}
	}()

	return waitForStreamInterruptible(ch, interruptCh)
}

func waitForStreamInterruptible(ch <-chan tea.Msg, interruptCh <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		select {
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			return msg
		case <-interruptCh:
			return InterruptMsg{Reason: "User interrupted"}
		}
	}
}

func compactHistory(history []llm.Message) tea.Cmd {
	return func() tea.Msg {
		messages := []llm.Message{
			{
				Role:    "system",
				Content: "You are a conversation summarizer. Produce a concise summary of the following conversation. Preserve key decisions, code changes, file paths, and important context. Output only the summary, no preamble.",
			},
		}
		// Append the full history as a single user message for context
		var sb strings.Builder
		for _, msg := range history {
			sb.WriteString("[" + msg.Role + "]: ")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		}
		messages = append(messages, llm.Message{
			Role:    "user",
			Content: sb.String(),
		})

		result, err := llm.CallLLM(messages, nil)
		if err != nil {
			log.Printf("compact error: %v", err)
			return CompactResultMsg{Err: err}
		}
		return CompactResultMsg{
			Summary: result.Delta.Content,
			Usage:   result.Usage,
		}
	}
}

func executeToolInterruptible(a *agent.Agent, tc llm.ToolCall, interruptCh <-chan struct{}) tea.Cmd {
	name := tc.Function.Name
	args := tc.Function.Arguments
	id := tc.ID

	return func() tea.Msg {
		done := make(chan tea.Msg, 1)
		go func() {
			result, err := a.ExecuteTool(name, args)
			if err != nil {
				log.Printf("tool error: %v", err)
				done <- ToolResultMsg{
					ToolCallID: id,
					ToolName:   name,
					Args:       args,
					Result:     err.Error(),
					Err:        err,
				}
				return
			}
			log.Printf("tool result: %.200s", result.Output)
			done <- ToolResultMsg{
				ToolCallID: id,
				ToolName:   name,
				Args:       args,
				Result:     result.Output,
			}
		}()

		if interruptCh == nil {
			return <-done
		}
		select {
		case msg := <-done:
			return msg
		case <-interruptCh:
			return InterruptMsg{Reason: "Operation interrupted by user"}
		}
	}
}
