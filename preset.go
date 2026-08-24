package testx

import "testing"

// FuncPreset adapts func(I) O into a reusable declarative subject.
type FuncPreset[I, O any] struct {
	fn func(I) O
}

// Func creates a preset for func(I) O.
func Func[I, O any](fn func(I) O) FuncPreset[I, O] {
	return FuncPreset[I, O]{fn: fn}
}

// Run executes cases with this preset.
func (p FuncPreset[I, O]) Run(t *testing.T, cases ...Case[I, O]) {
	t.Helper()
	Run(t, p.fn, cases...)
}

// FuncErrPreset adapts func(I) (O, error) into a reusable declarative subject.
type FuncErrPreset[I, O any] struct {
	fn func(I) (O, error)
}

// FuncErr creates a preset for the common Go result signature func(I) (O, error).
func FuncErr[I, O any](fn func(I) (O, error)) FuncErrPreset[I, O] {
	return FuncErrPreset[I, O]{fn: fn}
}

// Run executes cases with this preset, requiring no error unless a case says otherwise.
func (p FuncErrPreset[I, O]) Run(t *testing.T, cases ...Case[I, O]) {
	t.Helper()
	RunErr(t, p.fn, cases...)
}
