package testx

import "reflect"

// Panics succeeds when the actual value is a function that panics when called.
func (a Assertion[T]) Panics() bool {
	a.t.Helper()
	_, panicked, err := callAndRecover(a.actual)
	if err != "" {
		return a.fail("Panics %s", err)
	}
	if panicked {
		return true
	}
	return a.fail("expected function to panic")
}

// PanicsWith succeeds when the function panics with the expected value.
func (a Assertion[T]) PanicsWith(expected any) bool {
	a.t.Helper()
	value, panicked, err := callAndRecover(a.actual)
	if err != "" {
		return a.fail("PanicsWith %s", err)
	}
	if !panicked {
		return a.fail("expected function to panic with %#v", expected)
	}
	diff, panicValue := safeDiff(expected, value)
	if panicValue != nil {
		return a.fail("testx.PanicsWith could not compare panic values: %v", panicValue)
	}
	if diff != "" {
		return a.fail("panic values differ (-want +got):\n%s", diff)
	}
	return true
}

// NotPanics succeeds when the function returns normally.
func (a Assertion[T]) NotPanics() bool {
	a.t.Helper()
	value, panicked, err := callAndRecover(a.actual)
	if err != "" {
		return a.fail("NotPanics %s", err)
	}
	if !panicked {
		return true
	}
	return a.fail("expected function not to panic, got: %#v", value)
}

func callAndRecover(fn any) (panicValue any, panicked bool, errorMessage string) {
	value := reflect.ValueOf(fn)
	if !value.IsValid() || value.Kind() != reflect.Func || value.IsNil() {
		return nil, false, "requires a non-nil function"
	}
	typeOf := value.Type()
	if typeOf.NumIn() != 0 {
		return nil, false, "requires a function with no parameters"
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			panicValue = recovered
			panicked = true
		}
	}()
	value.Call(nil)
	return nil, false, ""
}
