package circuit

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota // Normal operation, requests pass through
	StateOpen                // Circuit is open, requests fail fast
	StateHalfOpen            // Probing state, allows one request to test if service recovered
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Config holds circuit breaker configuration
type Config struct {
	MaxFailures      int           // Number of consecutive failures to open the circuit
	SuccessThreshold int           // Number of successes needed in half-open to close the circuit
	OpenTimeout      time.Duration // Duration to wait before transitioning from OPEN to HALF_OPEN
}

// DefaultConfig returns sensible default circuit breaker configuration
func DefaultConfig() Config {
	return Config{
		MaxFailures:      5,
		SuccessThreshold: 2,
		OpenTimeout:      30 * time.Second,
	}
}

// Breaker implements the circuit breaker pattern
type Breaker struct {
	config Config
	
	state        atomic.Value // Stores State
	failures     atomic.Int32 // Current consecutive failure count
	successes    atomic.Int32 // Current consecutive success count (half-open)
	lastFailureTime atomic.Value // Stores time.Time when state transitioned to OPEN
	
	mu sync.RWMutex
}

// New creates a new circuit breaker with the given configuration
func New(config Config) *Breaker {
	b := &Breaker{
		config: config,
	}
	b.state.Store(StateClosed)
	return b
}

// NewDefault creates a circuit breaker with default configuration
func NewDefault() *Breaker {
	return New(DefaultConfig())
}

// State returns the current circuit breaker state
func (b *Breaker) State() State {
	return b.state.Load().(State)
}

// CanExecute returns whether the circuit breaker allows execution
func (b *Breaker) CanExecute() bool {
	return b.State() != StateOpen
}

// RecordSuccess records a successful operation
func (b *Breaker) RecordSuccess() {
	currentState := b.State()
	
	b.failures.Store(0)
	
	switch currentState {
	case StateClosed:
		// Reset success counter (not used in closed state)
		b.successes.Store(0)
		
	case StateHalfOpen:
		successes := b.successes.Add(1)
		if successes >= int32(b.config.SuccessThreshold) {
			b.setState(StateClosed)
			b.successes.Store(0)
		}
		
	case StateOpen:
		// Shouldn't happen, but handle gracefully
		b.setState(StateHalfOpen)
	}
}

// RecordFailure records a failed operation
func (b *Breaker) RecordFailure() {
	failures := b.failures.Add(1)
	currentState := b.State()
	
	b.successes.Store(0)
	
	switch currentState {
	case StateClosed:
		if failures >= int32(b.config.MaxFailures) {
			b.lastFailureTime.Store(time.Now())
			b.setState(StateOpen)
		}
		
	case StateHalfOpen:
		// Any failure in half-open opens the circuit again
		b.lastFailureTime.Store(time.Now())
		b.setState(StateOpen)
		
	case StateOpen:
		// Already open, update failure time
		b.lastFailureTime.Store(time.Now())
	}
}

// setState updates the circuit breaker state
func (b *Breaker) setState(state State) {
	b.state.Store(state)
}

// shouldAttemptReset returns true if the circuit should transition from OPEN to HALF_OPEN
func (b *Breaker) shouldAttemptReset() bool {
	if b.State() != StateOpen {
		return false
	}
	
	lastFailure, ok := b.lastFailureTime.Load().(time.Time)
	if !ok {
		return true
	}
	
	return time.Since(lastFailure) >= b.config.OpenTimeout
}

// Execute runs the given function through the circuit breaker
func (b *Breaker) Execute(ctx context.Context, fn func() error) error {
	// Check if we need to transition from OPEN to HALF_OPEN
	if b.shouldAttemptReset() {
		b.setState(StateHalfOpen)
	}
	
	// Fail fast if circuit is open
	if b.State() == StateOpen {
		return &OpenError{Timeout: b.getRemainingOpenTime()}
	}
	
	// Execute the function
	err := fn()
	
	if err == nil {
		b.RecordSuccess()
	} else {
		b.RecordFailure()
	}
	
	return err
}

// ExecuteWithResult runs the given function and returns its result through the circuit breaker
func ExecuteWithResult[T any](b *Breaker, ctx context.Context, fn func() (T, error)) (T, error) {
	var zero T

	// Check if we need to transition from OPEN to HALF_OPEN
	if b.shouldAttemptReset() {
		b.setState(StateHalfOpen)
	}

	// Fail fast if circuit is open
	if b.State() == StateOpen {
		return zero, &OpenError{Timeout: b.getRemainingOpenTime()}
	}

	// Execute the function
	result, err := fn()

	if err == nil {
		b.RecordSuccess()
	} else {
		b.RecordFailure()
	}

	return result, err
}

// getRemainingOpenTime returns how long until the circuit transitions to HALF_OPEN
func (b *Breaker) getRemainingOpenTime() time.Duration {
	lastFailure, ok := b.lastFailureTime.Load().(time.Time)
	if !ok {
		return 0
	}
	
	remaining := b.config.OpenTimeout - time.Since(lastFailure)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Reset manually resets the circuit breaker to CLOSED state
func (b *Breaker) Reset() {
	b.failures.Store(0)
	b.successes.Store(0)
	b.setState(StateClosed)
}

// GetStats returns current circuit breaker statistics
func (b *Breaker) GetStats() Stats {
	return Stats{
		State:            b.State(),
		ConsecutiveFailures: int(b.failures.Load()),
		ConsecutiveSuccesses: int(b.successes.Load()),
	}
}

// Stats represents circuit breaker statistics
type Stats struct {
	State               State
	ConsecutiveFailures int
	ConsecutiveSuccesses int
}

// OpenError is returned when the circuit is open
type OpenError struct {
	Timeout time.Duration // Time until circuit attempts to close
}

func (e *OpenError) Error() string {
	if e.Timeout > 0 {
		return fmt.Sprintf("circuit breaker is OPEN; retry in %v", e.Timeout.Round(time.Millisecond))
	}
	return "circuit breaker is OPEN"
}

// IsOpenError returns true if the error is an OpenError
func IsOpenError(err error) bool {
	_, ok := err.(*OpenError)
	return ok
}
