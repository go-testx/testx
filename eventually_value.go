package testx

import (
	"testing"
	"time"
)

// EventuallyValueAssertion polls a value-producing condition until it reports ready.
type EventuallyValueAssertion[T any] struct {
	t         testing.TB
	condition func() (T, bool)
	interval  time.Duration
	valid     bool
}

// EventuallyValue starts polling a func() (T, bool) condition.
func EventuallyValue[T any](t testing.TB, condition func() (T, bool)) EventuallyValueAssertion[T] {
	t.Helper()
	return EventuallyValueAssertion[T]{t: t, condition: condition, interval: 10 * time.Millisecond, valid: true}
}

// Every changes the polling interval.
func (e EventuallyValueAssertion[T]) Every(interval time.Duration) EventuallyValueAssertion[T] {
	if interval > 0 {
		e.interval = interval
	} else {
		e.valid = false
	}
	return e
}

// Within polls until the condition is ready or timeout expires.
func (e EventuallyValueAssertion[T]) Within(timeout time.Duration) (T, bool) {
	e.t.Helper()
	var zero T
	if e.condition == nil {
		e.t.Errorf("testx.EventuallyValue: condition is nil")
		return zero, false
	}
	if timeout <= 0 {
		e.t.Errorf("testx.EventuallyValue: timeout must be > 0")
		return zero, false
	}
	if !e.valid {
		e.t.Errorf("testx.EventuallyValue: interval must be > 0")
		return zero, false
	}

	deadline := time.Now().Add(timeout)
	for {
		value, ready := e.condition()
		if ready && !time.Now().After(deadline) {
			return value, true
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			e.t.Errorf("condition was not satisfied within %s", timeout)
			return zero, false
		}
		wait := e.interval
		if wait > remaining {
			wait = remaining
		}
		timer := time.NewTimer(wait)
		<-timer.C
	}
}
