package testx

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Assertion is a fluent assertion over an actual value.
type Assertion[T any] struct {
	t       testing.TB
	actual  T
	fatal   bool
	message string
}

// Assertions binds assertion helpers to one testing.TB.
//
// The bound methods accept any so a single value can be reused for different
// actual types. Use the package-level Assert/Require functions when preserving
// compile-time matching between actual and expected values is preferred.
type Assertions struct {
	t testing.TB
}

// New binds assertion helpers to t for the duration of a test.
func New(t testing.TB) Assertions {
	t.Helper()
	return Assertions{t: t}
}

// Assert creates a non-fatal assertion using the bound test.
func (a Assertions) Assert(actual any) Assertion[any] {
	a.t.Helper()
	return Assertion[any]{t: a.t, actual: actual}
}

// Require creates a fatal assertion using the bound test.
func (a Assertions) Require(actual any) Assertion[any] {
	a.t.Helper()
	return Assertion[any]{t: a.t, actual: actual, fatal: true}
}

// Assert creates a non-fatal assertion. A failed assertion reports with Errorf.
func Assert[T any](t testing.TB, actual T) Assertion[T] {
	t.Helper()
	return Assertion[T]{t: t, actual: actual}
}

// Require creates a fatal assertion. A failed assertion reports with Fatalf.
func Require[T any](t testing.TB, actual T) Assertion[T] {
	t.Helper()
	return Assertion[T]{t: t, actual: actual, fatal: true}
}

func (a Assertion[T]) fail(format string, args ...any) bool {
	a.t.Helper()
	if a.message != "" {
		format = a.message + ": " + format
	}
	if a.fatal {
		a.t.Fatalf(format, args...)
		return false
	}
	a.t.Errorf(format, args...)
	return false
}

// Because adds context to any failure reported by this assertion.
func (a Assertion[T]) Because(format string, args ...any) Assertion[T] {
	a.message = fmt.Sprintf(format, args...)
	return a
}

// Equal compares values using go-cmp and prints a -want +got diff.
func (a Assertion[T]) Equal(expected T, options ...cmp.Option) bool {
	a.t.Helper()

	diff, panicValue := safeDiff(expected, a.actual, options...)
	if panicValue != nil {
		return a.fail("testx.Equal could not compare values: %v\nHint: pass cmp options such as cmp.AllowUnexported or cmpopts.IgnoreUnexported when appropriate.", panicValue)
	}
	if diff != "" {
		return a.fail("values differ (-want +got):\n%s", diff)
	}
	return true
}

// NotEqual succeeds when go-cmp considers actual and unexpected different.
func (a Assertion[T]) NotEqual(unexpected T, options ...cmp.Option) bool {
	a.t.Helper()
	diff, panicValue := safeDiff(unexpected, a.actual, options...)
	if panicValue != nil {
		return a.fail("testx.NotEqual could not compare values: %v", panicValue)
	}
	if diff == "" {
		return a.fail("expected values to differ, but both were:\n%#v", a.actual)
	}
	return true
}

// Nil checks whether the actual value is nil, including typed nil values.
func (a Assertion[T]) Nil() bool {
	a.t.Helper()
	if isNil(a.actual) {
		return true
	}
	return a.fail("expected nil, got: %#v", a.actual)
}

// NotNil checks whether the actual value is non-nil, including typed nil values.
func (a Assertion[T]) NotNil() bool {
	a.t.Helper()
	if !isNil(a.actual) {
		return true
	}
	return a.fail("expected non-nil value")
}

// True checks that actual is the boolean true.
func (a Assertion[T]) True() bool {
	a.t.Helper()
	v, ok := any(a.actual).(bool)
	if !ok {
		return a.fail("True requires bool, got %T", a.actual)
	}
	if v {
		return true
	}
	return a.fail("expected true, got false")
}

// False checks that actual is the boolean false.
func (a Assertion[T]) False() bool {
	a.t.Helper()
	v, ok := any(a.actual).(bool)
	if !ok {
		return a.fail("False requires bool, got %T", a.actual)
	}
	if !v {
		return true
	}
	return a.fail("expected false, got true")
}

// Len checks length for strings, arrays, slices, maps and channels.
func (a Assertion[T]) Len(expected int) bool {
	a.t.Helper()
	v := reflect.ValueOf(a.actual)
	if !v.IsValid() {
		return a.fail("Len requires a sized value, got <nil>")
	}
	switch v.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
		if v.Len() == expected {
			return true
		}
		return a.fail("expected length %d, got %d", expected, v.Len())
	default:
		return a.fail("Len requires string/array/slice/map/chan, got %T", a.actual)
	}
}

// Empty checks the conventional zero-length/zero-value forms used in tests.
func (a Assertion[T]) Empty() bool {
	a.t.Helper()
	if isNil(a.actual) {
		return true
	}
	v := reflect.ValueOf(a.actual)
	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Chan:
		if v.Len() == 0 {
			return true
		}
	case reflect.Array:
		if v.IsZero() {
			return true
		}
	default:
		if v.IsZero() {
			return true
		}
	}
	return a.fail("expected empty value, got: %#v", a.actual)
}

// Contains supports string substring checks, array/slice membership and map keys.
func (a Assertion[T]) Contains(expected any) bool {
	a.t.Helper()

	v := reflect.ValueOf(a.actual)
	if v.IsValid() && v.Kind() == reflect.String {
		sub := reflect.ValueOf(expected)
		if !sub.IsValid() || sub.Kind() != reflect.String {
			return a.fail("Contains on string requires string needle, got %T", expected)
		}
		if strings.Contains(v.String(), sub.String()) {
			return true
		}
		return a.fail("expected %q to contain %q", v.String(), sub.String())
	}

	if v.IsValid() && (v.Kind() == reflect.Array || v.Kind() == reflect.Slice) {
		for i := 0; i < v.Len(); i++ {
			if reflect.DeepEqual(v.Index(i).Interface(), expected) {
				return true
			}
		}
		return a.fail("expected collection to contain %#v", expected)
	}
	if v.IsValid() && v.Kind() == reflect.Map {
		for _, key := range v.MapKeys() {
			if reflect.DeepEqual(key.Interface(), expected) {
				return true
			}
		}
		return a.fail("expected map to contain key %#v", expected)
	}

	return a.fail("Contains supports string/array/slice/map, got %T", a.actual)
}

// NotContains succeeds when a string, collection or map does not contain expected.
func (a Assertion[T]) NotContains(expected any) bool {
	a.t.Helper()
	v := reflect.ValueOf(a.actual)
	if v.IsValid() && v.Kind() == reflect.String {
		sub := reflect.ValueOf(expected)
		if !sub.IsValid() || sub.Kind() != reflect.String {
			return a.fail("NotContains on string requires string needle, got %T", expected)
		}
		if !strings.Contains(v.String(), sub.String()) {
			return true
		}
		return a.fail("expected %q not to contain %q", v.String(), sub.String())
	}
	if containsReflectValue(v, expected) {
		return a.fail("expected collection not to contain %#v", expected)
	}
	if v.IsValid() && (v.Kind() == reflect.Array || v.Kind() == reflect.Slice || v.Kind() == reflect.Map) {
		return true
	}
	return a.fail("NotContains supports string/array/slice/map, got %T", a.actual)
}

// ContainsAll succeeds when every expected element occurs in the actual collection.
func (a Assertion[T]) ContainsAll(expected any) bool {
	a.t.Helper()
	actual := reflect.ValueOf(a.actual)
	want := reflect.ValueOf(expected)
	if !isCollection(actual) || !isCollection(want) || actual.Kind() == reflect.Map || want.Kind() == reflect.Map {
		return a.fail("ContainsAll requires array/slice values, got %T and %T", a.actual, expected)
	}
	used := make([]bool, actual.Len())
	for i := 0; i < want.Len(); i++ {
		found := false
		for j := 0; j < actual.Len(); j++ {
			if !used[j] && reflect.DeepEqual(actual.Index(j).Interface(), want.Index(i).Interface()) {
				used[j] = true
				found = true
				break
			}
		}
		if !found {
			return a.fail("expected collection to contain all values, missing %#v", want.Index(i).Interface())
		}
	}
	return true
}

// ElementsMatch succeeds when two arrays or slices contain the same multiset of values.
func (a Assertion[T]) ElementsMatch(expected any) bool {
	a.t.Helper()
	actual := reflect.ValueOf(a.actual)
	want := reflect.ValueOf(expected)
	if !isCollection(actual) || !isCollection(want) || actual.Kind() == reflect.Map || want.Kind() == reflect.Map {
		return a.fail("ElementsMatch requires array/slice values, got %T and %T", a.actual, expected)
	}
	if actual.Len() != want.Len() {
		return a.fail("expected collections to have equal length, got %d and %d", want.Len(), actual.Len())
	}
	if !containsAllReflect(actual, want) || !containsAllReflect(want, actual) {
		return a.fail("expected collections to contain the same elements")
	}
	return true
}

func isCollection(v reflect.Value) bool {
	return v.IsValid() && (v.Kind() == reflect.Array || v.Kind() == reflect.Slice)
}

func containsAllReflect(actual, want reflect.Value) bool {
	used := make([]bool, actual.Len())
	for i := 0; i < want.Len(); i++ {
		found := false
		for j := 0; j < actual.Len(); j++ {
			if !used[j] && reflect.DeepEqual(actual.Index(j).Interface(), want.Index(i).Interface()) {
				used[j] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func containsReflectValue(v reflect.Value, expected any) bool {
	if !v.IsValid() {
		return false
	}
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if reflect.DeepEqual(v.Index(i).Interface(), expected) {
				return true
			}
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			if reflect.DeepEqual(key.Interface(), expected) {
				return true
			}
		}
	}
	return false
}

// MatchRegexp checks a string against a regular expression.
func (a Assertion[T]) MatchRegexp(pattern string) bool {
	a.t.Helper()
	s, ok := any(a.actual).(string)
	if !ok {
		return a.fail("MatchRegexp requires string, got %T", a.actual)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return a.fail("invalid regexp %q: %v", pattern, err)
	}
	if re.MatchString(s) {
		return true
	}
	return a.fail("expected %q to match regexp %q", s, pattern)
}

// NoError checks that actual is an error value and is nil.
func (a Assertion[T]) NoError() bool {
	a.t.Helper()
	err, ok := errorValue(a.actual)
	if !ok {
		return a.fail("NoError requires an error-compatible value, got %T", a.actual)
	}
	if err == nil {
		return true
	}
	return a.fail("expected no error, got: %v", err)
}

// Error checks that actual is a non-nil error.
func (a Assertion[T]) Error() bool {
	a.t.Helper()
	err, ok := errorValue(a.actual)
	if !ok {
		return a.fail("Error requires an error-compatible value, got %T", a.actual)
	}
	if err != nil {
		return true
	}
	return a.fail("expected an error, got nil")
}

// ErrorIs checks errors.Is(actual, target).
func (a Assertion[T]) ErrorIs(target error) bool {
	a.t.Helper()
	err, ok := errorValue(a.actual)
	if !ok {
		return a.fail("ErrorIs requires an error-compatible value, got %T", a.actual)
	}
	if errors.Is(err, target) {
		return true
	}
	return a.fail("expected errors.Is(err, %v), got: %v", target, err)
}

// ErrorAs checks errors.As(actual, target). target follows the same rules as errors.As.
func (a Assertion[T]) ErrorAs(target any) bool {
	a.t.Helper()
	err, ok := errorValue(a.actual)
	if !ok {
		return a.fail("ErrorAs requires an error-compatible value, got %T", a.actual)
	}
	matched, panicValue := safeErrorAs(err, target)
	if panicValue != nil {
		return a.fail("testx.ErrorAs could not inspect target: %v", panicValue)
	}
	if matched {
		return true
	}
	return a.fail("expected errors.As(err, %T) to succeed, got: %v", target, err)
}

// ErrorContains checks the text of a non-nil error.
func (a Assertion[T]) ErrorContains(substring string) bool {
	a.t.Helper()
	err, ok := errorValue(a.actual)
	if !ok {
		return a.fail("ErrorContains requires an error-compatible value, got %T", a.actual)
	}
	if err != nil && strings.Contains(err.Error(), substring) {
		return true
	}
	return a.fail("expected error containing %q, got: %v", substring, err)
}

func safeDiff[T any](want, got T, options ...cmp.Option) (diff string, panicValue any) {
	defer func() {
		panicValue = recover()
	}()
	return cmp.Diff(want, got, options...), nil
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

func errorValue(v any) (error, bool) {
	if v == nil {
		return nil, true
	}
	err, ok := v.(error)
	return err, ok
}

func safeErrorAs(err error, target any) (matched bool, panicValue any) {
	defer func() {
		panicValue = recover()
	}()
	return errors.As(err, target), nil
}
