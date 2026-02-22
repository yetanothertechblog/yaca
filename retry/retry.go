package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Config holds retry configuration parameters
type Config struct {
	MaxAttempts int           // Maximum number of retry attempts (including initial attempt)
	BaseDelay   time.Duration // Initial delay before first retry
	MaxDelay    time.Duration // Maximum delay between retries
	Jitter      float64       // Jitter factor (0.0 to 1.0) to add randomness to delays
}

// DefaultConfig returns a sensible default retry configuration
func DefaultConfig() Config {
	return Config{
		MaxAttempts: 3,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    10 * time.Second,
		Jitter:      0.1,
	}
}

// shouldRetry returns true if the error is retryable
func shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for common retryable error patterns
	errStr := err.Error()
	
	// Network errors
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporarily unavailable",
		"rate limit",
		"too many requests",
		"service unavailable",
		"gateway timeout",
		"bad gateway",
		"network is unreachable",
		"no such host",
		"connection timed out",
	}
	
	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}
	
	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || 
		 containsCaseInsensitive(s, substr))
}

func containsCaseInsensitive(s, substr string) bool {
	s, substr = toLower(s), toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

// calculateDelay computes the delay for a given attempt using exponential backoff with jitter
func calculateDelay(attempt int, config Config) time.Duration {
	// Exponential backoff: baseDelay * 2^(attempt-1)
	backoff := float64(config.BaseDelay) * math.Pow(2, float64(attempt-1))
	
	// Apply jitter to avoid thundering herd problem
	if config.Jitter > 0 {
		jitterRange := backoff * config.Jitter
		jitterOffset := (rand.Float64() - 0.5) * 2 * jitterRange
		backoff += jitterOffset
	}
	
	// Ensure delay is non-negative and doesn't exceed max delay
	if backoff < 0 {
		backoff = 0
	}
	if backoff > float64(config.MaxDelay) {
		backoff = float64(config.MaxDelay)
	}
	
	return time.Duration(backoff)
}

// Do executes a function with retry logic using exponential backoff
func Do(ctx context.Context, config Config, fn func() error) error {
	var lastErr error
	
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		if attempt > 0 {
			// Calculate delay before retry
			delay := calculateDelay(attempt, config)
			
			select {
			case <-time.After(delay):
				// Proceed with retry
			case <-ctx.Done():
				return fmt.Errorf("retry cancelled: %w", ctx.Err())
			}
		}
		
		// Execute the function
		err := fn()
		if err == nil {
			return nil // Success
		}
		
		lastErr = err
		
		// If this is not a retryable error or we've exhausted attempts, return
		if !shouldRetry(err) || attempt == config.MaxAttempts-1 {
			return fmt.Errorf("after %d attempt(s): %w", attempt+1, err)
		}
	}
	
	return lastErr
}

// DoWithResult executes a function with retry logic and returns a result
func DoWithResult[T any](ctx context.Context, cfg Config, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if attempt > 0 {
			delay := calculateDelay(attempt, cfg)

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return result, fmt.Errorf("retry cancelled: %w", ctx.Err())
			}
		}

		res, err := fn()
		if err == nil {
			return res, nil
		}

		result = res
		lastErr = err

		if !shouldRetry(err) || attempt == cfg.MaxAttempts-1 {
			return result, fmt.Errorf("after %d attempt(s): %w", attempt+1, err)
		}
	}

	return result, lastErr
}
