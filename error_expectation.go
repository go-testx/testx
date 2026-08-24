package testx

import (
	"errors"
	"fmt"
	"strings"
)

type errorExpectationKind uint8

const (
	expectNoError errorExpectationKind = iota
	expectAnyError
	expectErrorIs
	expectErrorContains
	expectErrorMatch
)

// ErrorExpectation describes what an error-returning preset should expect.
// Its zero value means "expect no error".
type ErrorExpectation struct {
	kind        errorExpectationKind
	target      error
	substring   string
	predicate   func(error) bool
	description string
}

// AnyError expects any non-nil error.
func AnyError() ErrorExpectation {
	return ErrorExpectation{kind: expectAnyError}
}

// ErrorIs expects errors.Is(actual, target) to be true.
func ErrorIs(target error) ErrorExpectation {
	return ErrorExpectation{kind: expectErrorIs, target: target}
}

// ErrorContains expects a non-nil error containing substring.
func ErrorContains(substring string) ErrorExpectation {
	return ErrorExpectation{kind: expectErrorContains, substring: substring}
}

// ErrorMatch expects predicate(actual) to be true.
func ErrorMatch(description string, predicate func(error) bool) ErrorExpectation {
	return ErrorExpectation{
		kind:        expectErrorMatch,
		predicate:   predicate,
		description: description,
	}
}

func (e ErrorExpectation) wantsError() bool {
	return e.kind != expectNoError
}

func (e ErrorExpectation) check(actual error) (bool, string) {
	switch e.kind {
	case expectNoError:
		if actual == nil {
			return true, ""
		}
		return false, fmt.Sprintf("expected no error, got: %v", actual)
	case expectAnyError:
		if actual != nil {
			return true, ""
		}
		return false, "expected an error, got nil"
	case expectErrorIs:
		if actual != nil && errors.Is(actual, e.target) {
			return true, ""
		}
		return false, fmt.Sprintf("expected errors.Is(err, %v), got: %v", e.target, actual)
	case expectErrorContains:
		if actual != nil && strings.Contains(actual.Error(), e.substring) {
			return true, ""
		}
		return false, fmt.Sprintf("expected error containing %q, got: %v", e.substring, actual)
	case expectErrorMatch:
		if actual != nil && e.predicate != nil {
			matched, panicValue := safeErrorPredicate(e.predicate, actual)
			if panicValue != nil {
				return false, fmt.Sprintf("error predicate panicked: %v", panicValue)
			}
			if matched {
				return true, ""
			}
		}
		desc := e.description
		if desc == "" {
			desc = "custom error predicate"
		}
		return false, fmt.Sprintf("expected %s, got: %v", desc, actual)
	default:
		return false, "invalid error expectation"
	}
}

func safeErrorPredicate(predicate func(error) bool, actual error) (matched bool, panicValue any) {
	defer func() {
		panicValue = recover()
	}()
	return predicate(actual), nil
}
