package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net"
	"strings"
	"time"
)

type Config struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Jitter      float64
}

func DefaultConfig() Config {
	return Config{
		MaxAttempts: 3,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    10 * time.Second,
		Jitter:      0.1,
	}
}

// ShouldRetry returns true if the error is transient and worth retrying.
func ShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Check for net.Error (timeout, temporary)
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}

	// Fall back to string matching for HTTP-level errors (e.g. "API error 429: ...")
	errStr := strings.ToLower(err.Error())
	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

var retryablePatterns = []string{
	"connection refused",
	"connection reset",
	"too many requests",
	"service unavailable",
	"gateway timeout",
	"bad gateway",
	"api error 429",
	"api error 502",
	"api error 503",
	"api error 504",
}

func calculateDelay(attempt int, cfg Config) time.Duration {
	backoff := float64(cfg.BaseDelay) * math.Pow(2, float64(attempt-1))
	if cfg.Jitter > 0 {
		jitter := backoff * cfg.Jitter * (rand.Float64()*2 - 1)
		backoff += jitter
	}
	backoff = max(backoff, 0)
	backoff = min(backoff, float64(cfg.MaxDelay))
	return time.Duration(backoff)
}

// Do executes fn with retries using exponential backoff. Non-retryable errors
// are returned immediately without wrapping.
func Do(ctx context.Context, cfg Config, fn func() error) error {
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if attempt > 0 {
			delay := calculateDelay(attempt, cfg)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return fmt.Errorf("retry cancelled: %w", ctx.Err())
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		if !ShouldRetry(err) || attempt == cfg.MaxAttempts-1 {
			if attempt > 0 {
				return fmt.Errorf("after %d attempts: %w", attempt+1, err)
			}
			return err
		}
	}
	return nil
}
