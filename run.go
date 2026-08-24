package testx

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Run executes cases against a simple func(I) O subject.
// It is the Level 3 convenience API; callers can always use Case values manually
// with testing.T.Run when they want IDE-visible table structure or custom flow.
func Run[I, O any](t *testing.T, fn func(I) O, cases ...Case[I, O]) {
	t.Helper()
	if fn == nil {
		t.Fatal("testx: subject function is nil")
	}
	runValue(t, fn, cases)
}

// RunErr executes cases against func(I) (O, error).
// By default each case expects no error. Use Case.WithError for error cases.
func RunErr[I, O any](t *testing.T, fn func(I) (O, error), cases ...Case[I, O]) {
	t.Helper()
	if fn == nil {
		t.Fatal("testx: subject function is nil")
	}
	runResult(t, fn, cases)
}

func runValue[I, O any](t *testing.T, fn func(I) O, cases []Case[I, O]) {
	t.Helper()
	if fn == nil {
		t.Fatal("testx: subject function is nil")
	}
	for i := range cases {
		c := cases[i]
		name := c.Name
		if name == "" {
			name = "case"
		}
		t.Run(name, func(t *testing.T) {
			if c.skip {
				t.Skip(c.skipReason)
			}
			if c.parallel {
				t.Parallel()
			}
			got := fn(c.Input)
			assertCaseEqual(t, c.Expect, got, c.cmpOptions)
		})
	}
}

func runResult[I, O any](t *testing.T, fn func(I) (O, error), cases []Case[I, O]) {
	t.Helper()
	if fn == nil {
		t.Fatal("testx: subject function is nil")
	}
	for i := range cases {
		c := cases[i]
		name := c.Name
		if name == "" {
			name = "case"
		}
		t.Run(name, func(t *testing.T) {
			if c.skip {
				t.Skip(c.skipReason)
			}
			if c.parallel {
				t.Parallel()
			}

			got, err := fn(c.Input)
			if ok, message := c.errExpect.check(err); !ok {
				t.Fatalf("%s", message)
			}

			// Error cases normally describe the error contract, not the partial output.
			if c.errExpect.wantsError() && !c.compareOnError {
				return
			}
			assertCaseEqual(t, c.Expect, got, c.cmpOptions)
		})
	}
}

func assertCaseEqual[O any](t testing.TB, want, got O, options []cmp.Option) {
	t.Helper()
	diff, panicValue := safeDiff(want, got, options...)
	if panicValue != nil {
		t.Fatalf("testx: go-cmp could not compare case output: %v", panicValue)
	}
	if diff != "" {
		t.Errorf("output differs (-want +got):\n%s", diff)
	}
}
