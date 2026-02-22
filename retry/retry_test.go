package retry

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"connection refused", errors.New("connection refused"), true},
		{"connection reset", errors.New("connection reset by peer"), true},
		{"timeout", errors.New("context timeout exceeded"), true},
		{"rate limit", errors.New("rate limit exceeded"), true},
		{"too many requests", errors.New("429 too many requests"), true},
		{"service unavailable", errors.New("503 service unavailable"), true},
		{"gateway timeout", errors.New("504 gateway timeout"), true},
		{"bad gateway", errors.New("502 bad gateway"), true},
		{"network unreachable", errors.New("network is unreachable"), true},
		{"no such host", errors.New("lookup: no such host"), true},
		{"non-retryable error", errors.New("bad request"), false},
		{"validation error", errors.New("invalid input"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldRetry(tt.err)
			if result != tt.expected {
				t.Errorf("shouldRetry(%q) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestCalculateDelay(t *testing.T) {
	config := Config{
		BaseDelay: 100 * time.Millisecond,
		MaxDelay:  5 * time.Second,
		Jitter:    0.0, // Disable jitter for predictable testing
	}

	tests := []struct {
		name           string
		attempt        int
		expectedMin    time.Duration
		expectedMax    time.Duration
	}{
		{"attempt 1", 1, 100 * time.Millisecond, 100 * time.Millisecond},
		{"attempt 2", 2, 200 * time.Millisecond, 200 * time.Millisecond},
		{"attempt 3", 3, 400 * time.Millisecond, 400 * time.Millisecond},
		{"attempt 4", 4, 800 * time.Millisecond, 800 * time.Millisecond},
		{"attempt 5", 5, 1600 * time.Millisecond, 1600 * time.Millisecond},
		{"attempt 6", 6, 3200 * time.Millisecond, 3200 * time.Millisecond},
		{"attempt 7 (capped)", 7, 5 * time.Second, 5 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := calculateDelay(tt.attempt, config)
			if delay < tt.expectedMin || delay > tt.expectedMax {
				t.Errorf("calculateDelay(%d) = %v, want between %v and %v",
					tt.attempt, delay, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestDo_Success(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Jitter:      0.1,
	}

	callCount := 0
	fn := func() error {
		callCount++
		return nil
	}

	err := Do(ctx, config, fn)

	if err != nil {
		t.Errorf("Do() returned error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Do() called function %d times, want 1", callCount)
	}
}

func TestDo_Retry(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Jitter:      0.0,
	}

	callCount := 0
	fn := func() error {
		callCount++
		if callCount < 2 {
			return errors.New("connection refused")
		}
		return nil
	}

	start := time.Now()
	err := Do(ctx, config, fn)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Do() returned error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("Do() called function %d times, want 2", callCount)
	}
	// Should have waited at least BaseDelay
	if elapsed < config.BaseDelay {
		t.Errorf("Do() completed in %v, want at least %v", elapsed, config.BaseDelay)
	}
}

func TestDo_MaxAttemptsExceeded(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Jitter:      0.0,
	}

	callCount := 0
	fn := func() error {
		callCount++
		return errors.New("connection refused")
	}

	err := Do(ctx, config, fn)

	if err == nil {
		t.Error("Do() expected error, got nil")
	}
	if callCount != 3 {
		t.Errorf("Do() called function %d times, want 3", callCount)
	}
}

func TestDo_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Jitter:      0.0,
	}

	callCount := 0
	fn := func() error {
		callCount++
		return errors.New("bad request")
	}

	err := Do(ctx, config, fn)

	if err == nil {
		t.Error("Do() expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("Do() called function %d times, want 1 (non-retryable)", callCount)
	}
}

func TestDo_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := Config{
		MaxAttempts: 10,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
		Jitter:      0.0,
	}

	callCount := 0
	fn := func() error {
		callCount++
		if callCount == 2 {
			cancel()
		}
		return errors.New("connection refused")
	}

	err := Do(ctx, config, fn)

	if err == nil {
		t.Error("Do() expected error due to cancellation, got nil")
	}
	if callCount != 2 {
		t.Errorf("Do() called function %d times, want 2", callCount)
	}
}

func TestDoWithResult_Success(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Jitter:      0.1,
	}

	callCount := 0
	fn := func() (string, error) {
		callCount++
		return "success", nil
	}

	result, err := DoWithResult(ctx, config, fn)

	if err != nil {
		t.Errorf("DoWithResult() returned error: %v", err)
	}
	if result != "success" {
		t.Errorf("DoWithResult() = %q, want %q", result, "success")
	}
	if callCount != 1 {
		t.Errorf("DoWithResult() called function %d times, want 1", callCount)
	}
}

func TestDoWithResult_Retry(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Jitter:      0.0,
	}

	callCount := 0
	fn := func() (string, error) {
		callCount++
		if callCount < 3 {
			return "", errors.New("timeout")
		}
		return "success", nil
	}

	result, err := DoWithResult(ctx, config, fn)

	if err != nil {
		t.Errorf("DoWithResult() returned error: %v", err)
	}
	if result != "success" {
		t.Errorf("DoWithResult() = %q, want %q", result, "success")
	}
	if callCount != 3 {
		t.Errorf("DoWithResult() called function %d times, want 3", callCount)
	}
}

func BenchmarkDo_Success(b *testing.B) {
	ctx := context.Background()
	config := DefaultConfig()
	fn := func() error { return nil }

	for i := 0; i < b.N; i++ {
		Do(ctx, config, fn)
	}
}

func ExampleConfig() {
	config := Config{
		MaxAttempts: 5,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    30 * time.Second,
		Jitter:      0.1,
	}

	ctx := context.Background()
	err := Do(ctx, config, func() error {
		// Your operation here
		return nil
	})

	if err != nil {
		fmt.Printf("Operation failed: %v\n", err)
	}
}
