package testx

import "github.com/google/go-cmp/cmp"

// Case describes a typed input/output test case.
//
// Case is intentionally useful at both abstraction levels:
//   - use it as plain table data with testing.T.Run;
//   - pass it to Run/RunErr for declarative execution.
type Case[I, O any] struct {
	Name   string
	Input  I
	Expect O

	cmpOptions     []cmp.Option
	errExpect      ErrorExpectation
	parallel       bool
	skip           bool
	skipReason     string
	compareOnError bool
}

// C creates a typed Case while allowing Go to infer I and O.
func C[I, O any](name string, input I, expect O) Case[I, O] {
	return Case[I, O]{Name: name, Input: input, Expect: expect}
}

// WithCmp adds go-cmp options used when this case is executed by testx.
func (c Case[I, O]) WithCmp(options ...cmp.Option) Case[I, O] {
	c.cmpOptions = append(append([]cmp.Option(nil), c.cmpOptions...), options...)
	return c
}

// WithError changes the expected error for RunErr / FuncErr presets.
// The zero-value expectation means "no error".
func (c Case[I, O]) WithError(expect ErrorExpectation) Case[I, O] {
	c.errExpect = expect
	return c
}

// CompareOutputOnError also compares the returned value when an expected error matches.
func (c Case[I, O]) CompareOutputOnError() Case[I, O] {
	c.compareOnError = true
	return c
}

// Parallel marks the generated subtest as parallel when executed by testx.
func (c Case[I, O]) Parallel() Case[I, O] {
	c.parallel = true
	return c
}

// Skip marks the generated subtest as skipped when executed by testx.
func (c Case[I, O]) Skip(reason string) Case[I, O] {
	c.skip = true
	c.skipReason = reason
	return c
}

// CaseSet is a reusable set of typed cases.
type CaseSet[I, O any] []Case[I, O]

// Cases groups cases into a reusable CaseSet.
func Cases[I, O any](cases ...Case[I, O]) CaseSet[I, O] {
	return append(CaseSet[I, O](nil), cases...)
}

// Run executes a reusable case set against func(I) O.
func (cases CaseSet[I, O]) Run(t TestT, fn func(I) O) {
	runValue(t, fn, cases)
}

// RunErr executes a reusable case set against func(I) (O, error).
func (cases CaseSet[I, O]) RunErr(t TestT, fn func(I) (O, error)) {
	runResult(t, fn, cases)
}
