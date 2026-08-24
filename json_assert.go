package testx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// JSONEqual compares JSON values semantically, ignoring whitespace and object key order.
func (a Assertion[T]) JSONEqual(expected any, options ...cmp.Option) bool {
	a.t.Helper()
	got, err := decodeJSONValue(a.actual)
	if err != nil {
		return a.fail("testx.JSONEqual could not decode actual JSON: %v", err)
	}
	want, err := decodeJSONValue(expected)
	if err != nil {
		return a.fail("testx.JSONEqual could not decode expected JSON: %v", err)
	}
	diff, panicValue := safeDiff(want, got, options...)
	if panicValue != nil {
		return a.fail("testx.JSONEqual could not compare values: %v", panicValue)
	}
	if diff != "" {
		return a.fail("JSON values differ (-want +got):\n%s", diff)
	}
	return true
}

// JSONEqual compares two JSON values using a non-fatal assertion.
func JSONEqual(t testing.TB, actual, expected any, options ...cmp.Option) bool {
	t.Helper()
	return Assert(t, actual).JSONEqual(expected, options...)
}

func decodeJSONValue(value any) (any, error) {
	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	case json.RawMessage:
		data = v
	default:
		var err error
		data, err = json.Marshal(value)
		if err != nil {
			return nil, err
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("multiple JSON values")
		}
		return nil, err
	}
	return normalizeJSONNumbers(decoded), nil
}

type canonicalJSONNumber string

func normalizeJSONNumbers(value any) any {
	switch v := value.(type) {
	case json.Number:
		number, ok := new(big.Rat).SetString(v.String())
		if !ok {
			return canonicalJSONNumber(v.String())
		}
		return canonicalJSONNumber(number.RatString())
	case []any:
		for i := range v {
			v[i] = normalizeJSONNumbers(v[i])
		}
	case map[string]any:
		for key := range v {
			v[key] = normalizeJSONNumbers(v[key])
		}
	}
	return value
}
