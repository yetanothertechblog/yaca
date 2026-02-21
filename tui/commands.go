package tui

import (
	"log"
	"strings"
	"sync/atomic"
	"time"

	"go-tui/agent"
	"go-tui/agent/tools"
	"go-tui/conversation"
	"go-tui/llm"

	tea "github.com/charmbracelet/bubbletea"
)

// Messages

type LLMResponseMsg struct {
	Content   string
	ToolCalls []llm.ToolCall
	Usage     *llm.Usage
	Err       error
}

type ToolResultMsg struct {
	ToolCallID string
	ToolName   string
	Args       string
	Result     string
	Err        error
}

// StreamTokenCountMsg is sent periodically during streaming to update the token counter.
type StreamTokenCountMsg struct {
	Count    int
	Thinking bool
	ch       <-chan tea.Msg
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

		var wordCount int64
		var thinking int32

		onContent := func(content string, isThinking bool) {
			if isThinking {
				atomic.StoreInt32(&thinking, 1)
			} else {
				atomic.StoreInt32(&thinking, 0)
			}
			words := int64(len(strings.Fields(content)))
			total := atomic.AddInt64(&wordCount, words)
			estimated := int(float64(total) * 0.75)
			select {
			case ch <- StreamTokenCountMsg{
				Count:    estimated,
				Thinking: atomic.LoadInt32(&thinking) == 1,
				ch:       ch,
			}:
			default:
			}
		}

		result, err := llm.CallLLMStream(messages, tools.All(), onContent)
		if err != nil {
			log.Printf("llm error: %v", err)
			ch <- LLMResponseMsg{Err: err}
			return
		}
		ch <- LLMResponseMsg{
			Content:   result.Delta.Content,
			ToolCalls: result.Delta.ToolCalls,
			Usage:     result.Usage,
		}
	}()

	return waitForStreamInterruptible(ch, interruptCh)
}

func waitForStreamInterruptible(ch <-chan tea.Msg, interruptCh <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(100 * time.Millisecond)
		var latest tea.Msg
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return latest
				}
				latest = msg
				if _, isToken := msg.(StreamTokenCountMsg); !isToken {
					return msg
				}
			case <-interruptCh:
				return InterruptMsg{Reason: "User interrupted"}
			default:
				if latest != nil {
					return latest
				}
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
