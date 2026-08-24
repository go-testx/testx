package testx

import (
	"testing"
	"time"
)

// EventuallyAssertion polls a condition until it succeeds or times out.
type EventuallyAssertion struct {
	t         testing.TB
	condition func() bool
	interval  time.Duration
	valid     bool
}

// Eventually starts an eventual assertion. The default poll interval is 10ms.
// Use Every before Within to customize polling.
func Eventually(t testing.TB, condition func() bool) EventuallyAssertion {
	t.Helper()
	return EventuallyAssertion{t: t, condition: condition, interval: 10 * time.Millisecond, valid: true}
}

// Every changes the polling interval.
func (e EventuallyAssertion) Every(interval time.Duration) EventuallyAssertion {
	if interval > 0 {
		e.interval = interval
	} else {
		e.valid = false
	}
	return e
}

// Within polls until condition returns true or timeout expires.
func (e EventuallyAssertion) Within(timeout time.Duration) bool {
	e.t.Helper()
	if e.condition == nil {
		e.t.Errorf("testx.Eventually: condition is nil")
		return false
	}
	if timeout <= 0 {
		e.t.Errorf("testx.Eventually: timeout must be > 0")
		return false
	}
	if !e.valid {
		e.t.Errorf("testx.Eventually: interval must be > 0")
		return false
	}

	deadline := time.Now().Add(timeout)
	for {
		if e.condition() && !time.Now().After(deadline) {
			return true
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			e.t.Errorf("condition was not satisfied within %s", timeout)
			return false
		}
		wait := e.interval
		if wait > remaining {
			wait = remaining
		}
		timer := time.NewTimer(wait)
		<-timer.C
	}
}
