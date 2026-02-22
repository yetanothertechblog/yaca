package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"429", &APIError{StatusCode: 429}, true},
		{"502", &APIError{StatusCode: 502}, true},
		{"503", &APIError{StatusCode: 503}, true},
		{"504", &APIError{StatusCode: 504}, true},
		{"400", &APIError{StatusCode: 400}, false},
		{"401", &APIError{StatusCode: 401}, false},
		{"404", &APIError{StatusCode: 404}, false},
		{"plain error", errors.New("something failed"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldRetry(tt.err); got != tt.want {
				t.Errorf("ShouldRetry(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestCalculateDelay(t *testing.T) {
	cfg := Config{BaseDelay: 100 * time.Millisecond, MaxDelay: 5 * time.Second, Jitter: 0}
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{7, 5 * time.Second}, // capped
	}
	for _, tt := range tests {
		if got := calculateDelay(tt.attempt, cfg); got != tt.want {
			t.Errorf("calculateDelay(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestDo_Success(t *testing.T) {
	calls := 0
	err := Do(context.Background(), Config{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}, func() error {
		calls++
		return nil
	})
	if err != nil || calls != 1 {
		t.Errorf("err=%v calls=%d", err, calls)
	}
}

func TestDo_RetriesThenSucceeds(t *testing.T) {
	calls := 0
	err := Do(context.Background(), Config{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}, func() error {
		calls++
		if calls < 2 {
			return &APIError{StatusCode: 503}
		}
		return nil
	})
	if err != nil || calls != 2 {
		t.Errorf("err=%v calls=%d", err, calls)
	}
}

func TestDo_ExhaustsAttempts(t *testing.T) {
	calls := 0
	err := Do(context.Background(), Config{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}, func() error {
		calls++
		return &APIError{StatusCode: 429}
	})
	if err == nil || calls != 3 {
		t.Errorf("err=%v calls=%d", err, calls)
	}
}

func TestDo_NonRetryableReturnsImmediately(t *testing.T) {
	calls := 0
	err := Do(context.Background(), Config{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}, func() error {
		calls++
		return &APIError{StatusCode: 400, Body: "bad request"}
	})
	if err == nil || calls != 1 {
		t.Errorf("err=%v calls=%d", err, calls)
	}
	// Non-retryable first-attempt errors should not be wrapped
	if err.Error() != "API error 400: bad request" {
		t.Errorf("err = %q, want unwrapped original", err.Error())
	}
}

func TestDo_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := Do(ctx, Config{MaxAttempts: 10, BaseDelay: 100 * time.Millisecond, MaxDelay: time.Second}, func() error {
		calls++
		if calls == 2 {
			cancel()
		}
		return &APIError{StatusCode: 502}
	})
	if err == nil || calls != 2 {
		t.Errorf("err=%v calls=%d", err, calls)
	}
}

func TestAPIError_Error(t *testing.T) {
	err := &APIError{StatusCode: 503, Body: "service unavailable"}
	if err.Error() != "API error 503: service unavailable" {
		t.Errorf("Error() = %q", err.Error())
	}
}

func BenchmarkDo_Success(b *testing.B) {
	cfg := DefaultConfig()
	fn := func() error { return nil }
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		Do(ctx, cfg, fn)
	}
}
