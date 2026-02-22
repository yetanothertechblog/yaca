package circuit

import (
	"fmt"
	"sync"
	"time"
)

type State int

const (
	StateClosed   State = iota
	StateOpen
	StateHalfOpen
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

type Config struct {
	MaxFailures      int
	SuccessThreshold int
	OpenTimeout      time.Duration
}

func DefaultConfig() Config {
	return Config{
		MaxFailures:      5,
		SuccessThreshold: 2,
		OpenTimeout:      30 * time.Second,
	}
}

type Breaker struct {
	config          Config
	mu              sync.Mutex
	state           State
	failures        int
	successes       int
	lastFailureTime time.Time
}

func New(config Config) *Breaker {
	return &Breaker{config: config, state: StateClosed}
}

func NewDefault() *Breaker {
	return New(DefaultConfig())
}

func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

func (b *Breaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	if b.state == StateHalfOpen {
		b.successes++
		if b.successes >= b.config.SuccessThreshold {
			b.state = StateClosed
			b.successes = 0
		}
	}
}

func (b *Breaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.successes = 0
	b.failures++
	switch b.state {
	case StateClosed:
		if b.failures >= b.config.MaxFailures {
			b.lastFailureTime = time.Now()
			b.state = StateOpen
		}
	case StateHalfOpen:
		b.lastFailureTime = time.Now()
		b.state = StateOpen
	case StateOpen:
		b.lastFailureTime = time.Now()
	}
}

// Execute runs fn through the circuit breaker, recording success/failure.
func (b *Breaker) Execute(fn func() error) error {
	b.mu.Lock()
	if b.state == StateOpen && time.Since(b.lastFailureTime) >= b.config.OpenTimeout {
		b.state = StateHalfOpen
	}
	if b.state == StateOpen {
		remaining := b.config.OpenTimeout - time.Since(b.lastFailureTime)
		b.mu.Unlock()
		return &OpenError{Timeout: max(remaining, 0)}
	}
	b.mu.Unlock()

	err := fn()
	if err == nil {
		b.RecordSuccess()
	} else {
		b.RecordFailure()
	}
	return err
}

func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.successes = 0
	b.state = StateClosed
}

func (b *Breaker) GetStats() Stats {
	b.mu.Lock()
	defer b.mu.Unlock()
	return Stats{
		State:               b.state,
		ConsecutiveFailures: b.failures,
	}
}

type Stats struct {
	State               State
	ConsecutiveFailures int
}

type OpenError struct {
	Timeout time.Duration
}

func (e *OpenError) Error() string {
	if e.Timeout > 0 {
		return fmt.Sprintf("circuit breaker is OPEN; retry in %v", e.Timeout.Round(time.Millisecond))
	}
	return "circuit breaker is OPEN"
}

func IsOpenError(err error) bool {
	_, ok := err.(*OpenError)
	return ok
}
