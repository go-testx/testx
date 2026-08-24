package testx

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestErrorExpectationChecks(t *testing.T) {
	sentinel := errors.New("sentinel")
	wrapped := errors.Join(errors.New("context"), sentinel)
	tests := []struct {
		name        string
		expectation ErrorExpectation
		actual      error
		wantOK      bool
		wantMessage string
	}{
		{name: "no error", expectation: ErrorExpectation{}, wantOK: true},
		{name: "unexpected error", expectation: ErrorExpectation{}, actual: sentinel, wantMessage: "expected no error"},
		{name: "any error", expectation: AnyError(), actual: sentinel, wantOK: true},
		{name: "missing any error", expectation: AnyError(), wantMessage: "expected an error"},
		{name: "error is", expectation: ErrorIs(sentinel), actual: wrapped, wantOK: true},
		{name: "error is mismatch", expectation: ErrorIs(sentinel), actual: errors.New("other"), wantMessage: "errors.Is"},
		{name: "contains", expectation: ErrorContains("tine"), actual: sentinel, wantOK: true},
		{name: "contains mismatch", expectation: ErrorContains("other"), actual: sentinel, wantMessage: "containing"},
		{name: "match", expectation: ErrorMatch("sentinel", func(err error) bool { return err == sentinel }), actual: sentinel, wantOK: true},
		{name: "match mismatch", expectation: ErrorMatch("sentinel", func(error) bool { return false }), actual: sentinel, wantMessage: "expected sentinel"},
		{name: "nil predicate", expectation: ErrorMatch("", nil), actual: sentinel, wantMessage: "custom error predicate"},
		{name: "panicking predicate", expectation: ErrorMatch("panic", func(error) bool { panic("boom") }), actual: sentinel, wantMessage: "predicate panicked"},
		{name: "invalid expectation", expectation: ErrorExpectation{kind: 255}, actual: sentinel, wantMessage: "invalid error expectation"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, message := tt.expectation.check(tt.actual)
			if ok != tt.wantOK {
				t.Fatalf("want ok %v, got %v (%s)", tt.wantOK, ok, message)
			}
			if tt.wantMessage != "" && !strings.Contains(message, tt.wantMessage) {
				t.Fatalf("want message containing %q, got %q", tt.wantMessage, message)
			}
		})
	}
}

func TestSafeHelpers(t *testing.T) {
	type private struct{ value int }
	if _, panicValue := safeDiff(private{value: 1}, private{value: 2}); panicValue == nil {
		t.Fatal("safeDiff should capture go-cmp panic")
	}
	if _, panicValue := safeErrorAs(errors.New("boom"), nil); panicValue == nil {
		t.Fatal("safeErrorAs should capture invalid target panic")
	}
	if matched, panicValue := safeErrorPredicate(func(error) bool { return true }, errors.New("boom")); !matched || panicValue != nil {
		t.Fatalf("unexpected predicate result: %v, %v", matched, panicValue)
	}
	if _, panicValue := safeErrorPredicate(func(error) bool { panic("boom") }, errors.New("boom")); panicValue == nil {
		t.Fatal("safeErrorPredicate should capture panic")
	}
	if _, ok := errorValue(1); ok {
		t.Fatal("integer should not be error-compatible")
	}
	var fn func()
	if !isNil(fn) || isNil(1) {
		t.Fatal("unexpected nil classification")
	}
}

func TestCollectionHelpers(t *testing.T) {
	actual := reflect.ValueOf([]int{1, 2})
	want := reflect.ValueOf([]int{2, 3})
	if containsAllReflect(actual, want) {
		t.Fatal("missing element should not match")
	}
	if !containsReflectValue(actual, 2) || containsReflectValue(actual, 3) {
		t.Fatal("unexpected slice membership")
	}
	if !containsReflectValue(reflect.ValueOf(map[string]int{"key": 1}), "key") {
		t.Fatal("map key should match")
	}
	if containsReflectValue(reflect.Value{}, "key") {
		t.Fatal("invalid value should not match")
	}
}

func TestJSONHelpers(t *testing.T) {
	decoded, err := decodeJSONValue(json.RawMessage(`{"n": 1e2}`))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"n": canonicalJSONNumber("100")}
	if !reflect.DeepEqual(decoded, want) {
		t.Fatalf("unexpected decoded JSON: %#v", decoded)
	}
	if _, err := decodeJSONValue(`{"broken"`); err == nil {
		t.Fatal("invalid JSON should fail")
	}
	if _, err := decodeJSONValue(`{} {}`); err == nil {
		t.Fatal("multiple JSON values should fail")
	}
	if _, err := decodeJSONValue(func() {}); err == nil {
		t.Fatal("unsupported Go value should fail JSON marshaling")
	}
	if got := normalizeJSONNumbers(json.Number("not-a-number")); got != canonicalJSONNumber("not-a-number") {
		t.Fatalf("unexpected invalid number normalization: %#v", got)
	}
}

func TestPanicHelpers(t *testing.T) {
	if _, _, message := callAndRecover(nil); message == "" {
		t.Fatal("nil function should be rejected")
	}
	if _, _, message := callAndRecover(func(int) {}); message == "" {
		t.Fatal("function with parameters should be rejected")
	}
	if _, panicked, message := callAndRecover(func() {}); panicked || message != "" {
		t.Fatalf("normal function returned unexpected result: %v, %q", panicked, message)
	}
	value, panicked, message := callAndRecover(func() { panic("boom") })
	if !panicked || value != "boom" || message != "" {
		t.Fatalf("panic was not captured: %#v, %v, %q", value, panicked, message)
	}
}

func TestGoldenHelpers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "value.golden")
	if err := writeFileAtomic(path, []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(path, []byte("second"), 0o644); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "second" {
		t.Fatalf("unexpected replacement content: %q", content)
	}
	if got := snapshotPath("Test/Child", ""); got != filepath.Join("testdata", "snapshots", "Test_Child_snapshot.golden") {
		t.Fatalf("unexpected snapshot path: %q", got)
	}
}
