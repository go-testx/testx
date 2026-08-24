package testx

import (
	"context"
	"testing"
)

// ContextFuncErrPreset adapts func(context.Context, I) (O, error) into a reusable subject.
type ContextFuncErrPreset[I, O any] struct {
	fn func(context.Context, I) (O, error)
}

// ContextFuncErr creates a preset for context-aware functions.
func ContextFuncErr[I, O any](fn func(context.Context, I) (O, error)) ContextFuncErrPreset[I, O] {
	return ContextFuncErrPreset[I, O]{fn: fn}
}

// Run executes each case with a fresh cancellable background context.
func (p ContextFuncErrPreset[I, O]) Run(t *testing.T, cases ...Case[I, O]) {
	t.Helper()
	if p.fn == nil {
		t.Fatal("testx: context subject function is nil")
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
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			got, err := p.fn(ctx, c.Input)
			if ok, message := c.errExpect.check(err); !ok {
				t.Fatalf("%s", message)
			}
			if c.errExpect.wantsError() && !c.compareOnError {
				return
			}
			assertCaseEqual(t, c.Expect, got, c.cmpOptions)
		})
	}
}
