package testx

import "testing"

// Factory constructs a fresh implementation for a contract spec.
// It receives the real *testing.T so implementations can use TempDir, Cleanup, etc.
type Factory[T any] func(*testing.T) T

// Spec describes one behavioral requirement of a contract.
type Spec[T any] struct {
	Name string
	Test func(*testing.T, T)
}

// S creates a contract Spec with type inference.
func S[T any](name string, test func(*testing.T, T)) Spec[T] {
	return Spec[T]{Name: name, Test: test}
}

// Contract is a reusable behavior suite, useful for testing multiple interface implementations.
type Contract[T any] struct {
	Name  string
	Specs []Spec[T]
}

// NewContract creates a reusable contract suite.
func NewContract[T any](name string, specs ...Spec[T]) Contract[T] {
	return Contract[T]{Name: name, Specs: append([]Spec[T](nil), specs...)}
}

// Implementation binds a name to a factory for VerifyAll.
type Implementation[T any] struct {
	Name    string
	Factory Factory[T]
}

// Impl creates a named contract implementation.
func Impl[T any](name string, factory Factory[T]) Implementation[T] {
	return Implementation[T]{Name: name, Factory: factory}
}

// Verify runs this contract against one implementation.
func (c Contract[T]) Verify(t *testing.T, factory Factory[T]) {
	t.Helper()
	c.verifySpecs(t, factory)
}

// VerifyAs adds an implementation-name subtest around Verify.
func (c Contract[T]) VerifyAs(t *testing.T, name string, factory Factory[T]) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		c.verifySpecs(t, factory)
	})
}

// VerifyAll runs the contract against each implementation and creates a test tree
// implementation -> spec.
func (c Contract[T]) VerifyAll(t *testing.T, implementations ...Implementation[T]) {
	t.Helper()
	for _, impl := range implementations {
		impl := impl
		t.Run(impl.Name, func(t *testing.T) {
			c.verifySpecs(t, impl.Factory)
		})
	}
}

func (c Contract[T]) verifySpecs(t *testing.T, factory Factory[T]) {
	t.Helper()
	if factory == nil {
		t.Fatalf("testx: contract %q has nil factory", c.Name)
	}
	for _, spec := range c.Specs {
		spec := spec
		t.Run(spec.Name, func(t *testing.T) {
			if spec.Test == nil {
				t.Fatalf("testx: contract %q has nil spec test", c.Name)
			}
			subject := factory(t)
			spec.Test(t, subject)
		})
	}
}
