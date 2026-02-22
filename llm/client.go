package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"go-tui/circuit"
	"go-tui/config"
	"go-tui/retry"
)

var (
	mu          sync.RWMutex
	activeURL   string
	activeKey   string
	activeModel string
	breaker     = circuit.NewDefault()
)

// Configure sets the active model configuration used by CallLLM and CallLLMStream.
func Configure(apiURL, apiKey, model string) {
	mu.Lock()
	defer mu.Unlock()
	activeURL = apiURL
	activeKey = apiKey
	activeModel = model
}

func getConfig() (string, string, string) {
	mu.RLock()
	defer mu.RUnlock()
	return activeURL, activeKey, activeModel
}

func GetCircuitBreakerStats() circuit.Stats {
	return breaker.GetStats()
}

func ResetCircuitBreaker() {
	breaker.Reset()
}

var retryCfg = retry.Config{
	MaxAttempts: config.MaxRetryAttempts,
	BaseDelay:   config.RetryBaseDelay,
	MaxDelay:    config.RetryMaxDelay,
	Jitter:      config.RetryJitter,
}

func CallLLM(messages []Message, tools []Tool) (*LLMResult, error) {
	var result *LLMResult
	err := breaker.Execute(func() error {
		return retry.Do(context.Background(), retryCfg, func() error {
			var err error
			result, err = doCallLLM(messages, tools)
			return err
		})
	})
	return result, err
}

func doCallLLM(messages []Message, tools []Tool) (*LLMResult, error) {
	url, key, model := getConfig()
	if url == "" || key == "" || model == "" {
		return nil, fmt.Errorf("LLM not configured: call Configure() first")
	}

	req := ChatRequest{
		Model:    model,
		Messages: messages,
		Tools:    tools,
		Stream:   false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+key)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody bytes.Buffer
		errBody.ReadFrom(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, errBody.String())
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &LLMResult{
		Delta: chatResp.Choices[0].Message,
		Usage: chatResp.Usage,
	}, nil
}

// CallLLMStream uses the circuit breaker but does NOT retry, since partial
// content may have already been streamed to the caller.
func CallLLMStream(messages []Message, tools []Tool, onContent func(string, bool)) (*LLMResult, error) {
	var result *LLMResult
	err := breaker.Execute(func() error {
		var err error
		result, err = doCallLLMStream(messages, tools, onContent)
		return err
	})
	return result, err
}

func doCallLLMStream(messages []Message, tools []Tool, onContent func(string, bool)) (*LLMResult, error) {
	url, key, model := getConfig()
	if url == "" || key == "" || model == "" {
		return nil, fmt.Errorf("LLM not configured: call Configure() first")
	}

	req := ChatRequest{
		Model:    model,
		Messages: messages,
		Tools:    tools,
		Stream:   true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+key)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody bytes.Buffer
		errBody.ReadFrom(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, errBody.String())
	}

	full := &Delta{}
	var usage *Usage
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk ChatResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if chunk.Usage != nil {
			usage = chunk.Usage
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta
		if delta == nil {
			continue
		}

		if delta.ReasoningContent != "" {
			full.ReasoningContent += delta.ReasoningContent
			onContent(delta.ReasoningContent, true)
		}
		if delta.Content != "" {
			full.Content += delta.Content
			onContent(delta.Content, false)
		}
		if len(delta.ToolCalls) > 0 {
			for _, tc := range delta.ToolCalls {
				if tc.ID != "" {
					full.ToolCalls = append(full.ToolCalls, tc)
				} else if len(full.ToolCalls) > 0 {
					last := &full.ToolCalls[len(full.ToolCalls)-1]
					last.Function.Arguments += tc.Function.Arguments
				}
			}
		}
	}

	return &LLMResult{Delta: full, Usage: usage}, scanner.Err()
}
