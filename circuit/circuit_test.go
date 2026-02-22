package circuit

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	config := Config{
		MaxFailures:      3,
		SuccessThreshold: 2,
		OpenTimeout:      10 * time.Second,
	}

	breaker := New(config)

	if breaker.State() != StateClosed {
		t.Errorf("New() State = %v, want CLOSED", breaker.State())
	}

	stats := breaker.GetStats()
	if stats.State != StateClosed {
		t.Errorf("New() Stats.State = %v, want CLOSED", stats.State)
	}
	if stats.ConsecutiveFailures != 0 {
		t.Errorf("New() Stats.ConsecutiveFailures = %d, want 0", stats.ConsecutiveFailures)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxFailures != 5 {
		t.Errorf("DefaultConfig().MaxFailures = %d, want 5", config.MaxFailures)
	}
	if config.SuccessThreshold != 2 {
		t.Errorf("DefaultConfig().SuccessThreshold = %d, want 2", config.SuccessThreshold)
	}
	if config.OpenTimeout != 30*time.Second {
		t.Errorf("DefaultConfig().OpenTimeout = %v, want 30s", config.OpenTimeout)
	}
}

func TestNewDefault(t *testing.T) {
	breaker := NewDefault()

	if breaker.State() != StateClosed {
		t.Errorf("NewDefault() State = %v, want CLOSED", breaker.State())
	}
}

func TestRecordSuccess(t *testing.T) {
	breaker := NewDefault()

	// Record some failures first
	for i := 0; i < 3; i++ {
		breaker.RecordFailure()
	}

	stats := breaker.GetStats()
	if stats.ConsecutiveFailures != 3 {
		t.Errorf("After 3 failures, ConsecutiveFailures = %d, want 3", stats.ConsecutiveFailures)
	}

	// Record success
	breaker.RecordSuccess()

	stats = breaker.GetStats()
	if stats.ConsecutiveFailures != 0 {
		t.Errorf("After success, ConsecutiveFailures = %d, want 0", stats.ConsecutiveFailures)
	}
}

func TestRecordFailure(t *testing.T) {
	config := Config{
		MaxFailures:      3,
		SuccessThreshold: 2,
		OpenTimeout:      1 * time.Second,
	}
	breaker := New(config)

	// Record failures until circuit opens
	for i := 0; i < 3; i++ {
		breaker.RecordFailure()
	}

	if breaker.State() != StateOpen {
		t.Errorf("After 3 failures, State = %v, want OPEN", breaker.State())
	}
}

func TestExecute_Success(t *testing.T) {
	ctx := context.Background()
	breaker := NewDefault()
	callCount := 0

	fn := func() error {
		callCount++
		return nil
	}

	err := breaker.Execute(ctx, fn)

	if err != nil {
		t.Errorf("Execute() returned error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Execute() called function %d times, want 1", callCount)
	}
	if breaker.State() != StateClosed {
		t.Errorf("Execute() State = %v, want CLOSED", breaker.State())
	}
}

func TestExecute_Failure(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxFailures:      3,
		SuccessThreshold: 2,
		OpenTimeout:      1 * time.Second,
	}
	breaker := New(config)

	fn := func() error {
		return errors.New("operation failed")
	}

	// Execute failures until circuit opens
	for i := 0; i < 3; i++ {
		err := breaker.Execute(ctx, fn)
		if err == nil {
			t.Errorf("Execute() expected error, got nil")
		}
	}

	if breaker.State() != StateOpen {
		t.Errorf("After 3 failures, State = %v, want OPEN", breaker.State())
	}
}

func TestExecute_Open(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxFailures:      2,
		SuccessThreshold: 1,
		OpenTimeout:      1 * time.Second,
	}
	breaker := New(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		breaker.RecordFailure()
	}

	// Try to execute while circuit is open
	callCount := 0
	fn := func() error {
		callCount++
		return nil
	}

	err := breaker.Execute(ctx, fn)

	if err == nil {
		t.Error("Execute() expected error when circuit is open, got nil")
	}
	if !IsOpenError(err) {
		t.Errorf("Execute() error type = %T, want *OpenError", err)
	}
	if callCount != 0 {
		t.Errorf("Execute() called function %d times when circuit is open, want 0", callCount)
	}
}

func TestExecute_HalfOpen(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxFailures:      2,
		SuccessThreshold: 2,
		OpenTimeout:      100 * time.Millisecond,
	}
	breaker := New(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		breaker.RecordFailure()
	}

	if breaker.State() != StateOpen {
		t.Errorf("Initial State = %v, want OPEN", breaker.State())
	}

	// Wait for open timeout to pass
	time.Sleep(150 * time.Millisecond)

	// First success in half-open state
	callCount := 0
	fn := func() error {
		callCount++
		return nil
	}

	err := breaker.Execute(ctx, fn)
	if err != nil {
		t.Errorf("First Execute() error: %v", err)
	}

	if breaker.State() != StateHalfOpen {
		t.Errorf("After 1 success, State = %v, want HALF_OPEN", breaker.State())
	}

	// Second success should close the circuit
	err = breaker.Execute(ctx, fn)
	if err != nil {
		t.Errorf("Second Execute() error: %v", err)
	}

	if breaker.State() != StateClosed {
		t.Errorf("After 2 successes, State = %v, want CLOSED", breaker.State())
	}
}

func TestExecute_HalfOpenFailure(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxFailures:      2,
		SuccessThreshold: 2,
		OpenTimeout:      100 * time.Millisecond,
	}
	breaker := New(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		breaker.RecordFailure()
	}

	// Wait for open timeout to pass
	time.Sleep(150 * time.Millisecond)

	// First success in half-open
	fnSuccess := func() error {
		return nil
	}
	err := breaker.Execute(ctx, fnSuccess)
	if err != nil {
		t.Errorf("First Execute() error: %v", err)
	}

	// Second call fails, should open circuit again
	fnFail := func() error {
		return errors.New("operation failed")
	}
	err = breaker.Execute(ctx, fnFail)
	if err == nil {
		t.Error("Second Execute() expected error, got nil")
	}

	if breaker.State() != StateOpen {
		t.Errorf("After failure in half-open, State = %v, want OPEN", breaker.State())
	}
}

func TestExecuteWithResult(t *testing.T) {
	ctx := context.Background()
	breaker := NewDefault()
	callCount := 0

	fn := func() (string, error) {
		callCount++
		return "result", nil
	}

	result, err := ExecuteWithResult(breaker, ctx, fn)

	if err != nil {
		t.Errorf("ExecuteWithResult() returned error: %v", err)
	}
	if result != "result" {
		t.Errorf("ExecuteWithResult() = %q, want %q", result, "result")
	}
	if callCount != 1 {
		t.Errorf("ExecuteWithResult() called function %d times, want 1", callCount)
	}
}

func TestExecuteWithResult_Failure(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxFailures:      2,
		SuccessThreshold: 2,
		OpenTimeout:      1 * time.Second,
	}
	breaker := New(config)

	fn := func() (string, error) {
		return "", errors.New("operation failed")
	}

	// Execute failures until circuit opens
	for i := 0; i < 2; i++ {
		_, err := ExecuteWithResult(breaker, ctx, fn)
		if err == nil {
			t.Errorf("ExecuteWithResult() expected error, got nil")
		}
	}

	if breaker.State() != StateOpen {
		t.Errorf("After 2 failures, State = %v, want OPEN", breaker.State())
	}
}

func TestExecuteWithResult_Open(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxFailures:      2,
		SuccessThreshold: 1,
		OpenTimeout:      1 * time.Second,
	}
	breaker := New(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		breaker.RecordFailure()
	}

	// Try to execute while circuit is open
	callCount := 0
	fn := func() (string, error) {
		callCount++
		return "result", nil
	}

	_, err := ExecuteWithResult(breaker, ctx, fn)

	if err == nil {
		t.Error("ExecuteWithResult() expected error when circuit is open, got nil")
	}
	if !IsOpenError(err) {
		t.Errorf("ExecuteWithResult() error type = %T, want *OpenError", err)
	}
	if callCount != 0 {
		t.Errorf("ExecuteWithResult() called function %d times when circuit is open, want 0", callCount)
	}
}

func TestReset(t *testing.T) {
	config := Config{
		MaxFailures:      2,
		SuccessThreshold: 1,
		OpenTimeout:      1 * time.Second,
	}
	breaker := New(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		breaker.RecordFailure()
	}

	if breaker.State() != StateOpen {
		t.Errorf("Before reset, State = %v, want OPEN", breaker.State())
	}

	// Reset the breaker
	breaker.Reset()

	if breaker.State() != StateClosed {
		t.Errorf("After reset, State = %v, want CLOSED", breaker.State())
	}

	stats := breaker.GetStats()
	if stats.ConsecutiveFailures != 0 {
		t.Errorf("After reset, ConsecutiveFailures = %d, want 0", stats.ConsecutiveFailures)
	}
	if stats.ConsecutiveSuccesses != 0 {
		t.Errorf("After reset, ConsecutiveSuccesses = %d, want 0", stats.ConsecutiveSuccesses)
	}
}

func TestOpenError(t *testing.T) {
	err := &OpenError{Timeout: 5 * time.Second}

	errStr := err.Error()
	if errStr == "" {
		t.Error("OpenError.Error() returned empty string")
	}
}

func TestIsOpenError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"OpenError", &OpenError{}, true},
		{"standard error", errors.New("some error"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsOpenError(tt.err)
			if result != tt.expected {
				t.Errorf("IsOpenError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func BenchmarkExecute_Success(b *testing.B) {
	ctx := context.Background()
	breaker := NewDefault()
	fn := func() error { return nil }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		breaker.Execute(ctx, fn)
	}
}

func ExampleBreaker_Execute() {
	breaker := NewDefault()
	ctx := context.Background()

	err := breaker.Execute(ctx, func() error {
		// Your operation here
		return nil
	})

	if err != nil {
		// Handle error
	}
}

func ExampleExecuteWithResult() {
	breaker := NewDefault()
	ctx := context.Background()

	result, err := ExecuteWithResult(breaker, ctx, func() (string, error) {
		// Your operation here
		return "success", nil
	})

	if err != nil {
		// Handle error
	}

	_ = result
}
