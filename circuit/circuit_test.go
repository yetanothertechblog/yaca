package circuit

import (
	"errors"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	breaker := New(Config{MaxFailures: 3, SuccessThreshold: 2, OpenTimeout: 10 * time.Second})
	if breaker.State() != StateClosed {
		t.Errorf("State = %v, want CLOSED", breaker.State())
	}
	stats := breaker.GetStats()
	if stats.ConsecutiveFailures != 0 {
		t.Errorf("ConsecutiveFailures = %d, want 0", stats.ConsecutiveFailures)
	}
}

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	if c.MaxFailures != 5 || c.SuccessThreshold != 2 || c.OpenTimeout != 30*time.Second {
		t.Errorf("DefaultConfig() = %+v", c)
	}
}

func TestRecordSuccess_ResetsFailures(t *testing.T) {
	b := NewDefault()
	for i := 0; i < 3; i++ {
		b.RecordFailure()
	}
	b.RecordSuccess()
	if s := b.GetStats(); s.ConsecutiveFailures != 0 {
		t.Errorf("ConsecutiveFailures = %d, want 0", s.ConsecutiveFailures)
	}
}

func TestRecordFailure_OpensCircuit(t *testing.T) {
	b := New(Config{MaxFailures: 3, SuccessThreshold: 2, OpenTimeout: time.Second})
	for i := 0; i < 3; i++ {
		b.RecordFailure()
	}
	if b.State() != StateOpen {
		t.Errorf("State = %v, want OPEN", b.State())
	}
}

func TestExecute_Success(t *testing.T) {
	b := NewDefault()
	called := false
	err := b.Execute(func() error { called = true; return nil })
	if err != nil || !called {
		t.Errorf("Execute() err=%v called=%v", err, called)
	}
}

func TestExecute_OpensAfterFailures(t *testing.T) {
	b := New(Config{MaxFailures: 3, SuccessThreshold: 2, OpenTimeout: time.Second})
	for i := 0; i < 3; i++ {
		b.Execute(func() error { return errors.New("fail") })
	}
	if b.State() != StateOpen {
		t.Errorf("State = %v, want OPEN", b.State())
	}
}

func TestExecute_FailsFastWhenOpen(t *testing.T) {
	b := New(Config{MaxFailures: 2, SuccessThreshold: 1, OpenTimeout: time.Second})
	for i := 0; i < 2; i++ {
		b.RecordFailure()
	}
	called := false
	err := b.Execute(func() error { called = true; return nil })
	if !IsOpenError(err) {
		t.Errorf("err type = %T, want *OpenError", err)
	}
	if called {
		t.Error("function should not be called when circuit is open")
	}
}

func TestExecute_HalfOpenRecovery(t *testing.T) {
	b := New(Config{MaxFailures: 2, SuccessThreshold: 2, OpenTimeout: 100 * time.Millisecond})
	for i := 0; i < 2; i++ {
		b.RecordFailure()
	}
	time.Sleep(150 * time.Millisecond)

	// First success: still half-open
	b.Execute(func() error { return nil })
	if b.State() != StateHalfOpen {
		t.Errorf("State = %v, want HALF_OPEN", b.State())
	}

	// Second success: closes
	b.Execute(func() error { return nil })
	if b.State() != StateClosed {
		t.Errorf("State = %v, want CLOSED", b.State())
	}
}

func TestExecute_HalfOpenFailureReopens(t *testing.T) {
	b := New(Config{MaxFailures: 2, SuccessThreshold: 2, OpenTimeout: 100 * time.Millisecond})
	for i := 0; i < 2; i++ {
		b.RecordFailure()
	}
	time.Sleep(150 * time.Millisecond)

	b.Execute(func() error { return nil })
	b.Execute(func() error { return errors.New("fail") })
	if b.State() != StateOpen {
		t.Errorf("State = %v, want OPEN", b.State())
	}
}

func TestReset(t *testing.T) {
	b := New(Config{MaxFailures: 2, SuccessThreshold: 1, OpenTimeout: time.Second})
	for i := 0; i < 2; i++ {
		b.RecordFailure()
	}
	b.Reset()
	if b.State() != StateClosed {
		t.Errorf("State = %v, want CLOSED", b.State())
	}
	if s := b.GetStats(); s.ConsecutiveFailures != 0 {
		t.Errorf("ConsecutiveFailures = %d, want 0", s.ConsecutiveFailures)
	}
}

func TestOpenError(t *testing.T) {
	err := &OpenError{Timeout: 5 * time.Second}
	if err.Error() == "" {
		t.Error("OpenError.Error() returned empty string")
	}
}

func TestIsOpenError(t *testing.T) {
	if !IsOpenError(&OpenError{}) {
		t.Error("IsOpenError(OpenError) = false")
	}
	if IsOpenError(errors.New("x")) {
		t.Error("IsOpenError(plain error) = true")
	}
	if IsOpenError(nil) {
		t.Error("IsOpenError(nil) = true")
	}
}

func BenchmarkExecute_Success(b *testing.B) {
	breaker := NewDefault()
	fn := func() error { return nil }
	for i := 0; i < b.N; i++ {
		breaker.Execute(fn)
	}
}
