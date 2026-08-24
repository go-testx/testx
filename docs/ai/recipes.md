# testx Test-Generation Recipes

Use these patterns as starting points. Preserve project conventions, choose meaningful fixtures, and adapt names to the domain.

## Value plus error

```go
func TestParse(t *testing.T) {
    testx.RunErr(t, Parse,
        testx.C("valid info", "INFO ready", Entry{
            Level:   "INFO",
            Message: "ready",
        }),
        testx.C("invalid level", "BAD ready", Entry{}).
            WithError(testx.ErrorContains("invalid level")),
    )
}
```

Choose `ErrorIs` for sentinel/wrapped errors, `ErrorContains` only when message text is the public contract, `AnyError` when only non-nil matters, and `ErrorMatch` for a necessary custom predicate.

## Compare selected fields

Do not compare an entire result if only selected fields are contractual:

```go
func TestParser_Parse(t *testing.T) {
    got, err := parser.Parse(raw)
    testx.Require(t, err).NoError()
    testx.Assert(t, got.Level).Equal(want.Level)
    testx.Assert(t, got.Message).Equal(want.Message)
}
```

Use `cmpopts.IgnoreFields` only when whole-object comparison remains clearer than explicit field assertions.

## Hand-written table with custom flow

```go
func TestNormalize(t *testing.T) {
    cases := []testx.Case[string, string]{
        testx.C("trim", " value ", "value"),
        testx.C("already clean", "value", "value"),
    }

    for _, tc := range cases {
        tc := tc
        t.Run(tc.Name, func(t *testing.T) {
            got := Normalize(tc.Input)
            testx.Assert(t, got).Equal(tc.Expect)
        })
    }
}
```

Use this level when each Case needs custom setup, multiple calls, intermediate assertions, or cleanup.

## Reuse a CaseSet

```go
var normalizeCases = testx.Cases(
    testx.C("trim", " value ", "value"),
    testx.C("empty", "", ""),
)

func TestNormalize(t *testing.T) {
    normalizeCases.Run(t, Normalize)
}
```

Keep package-level CaseSets immutable. Prefer literal/static names for predictable subtest paths and IDE support.

## Context-aware function

```go
func TestService_Load(t *testing.T) {
    testx.ContextFuncErr(service.Load).Run(t,
        testx.C("existing", "42", Item{ID: "42"}),
        testx.C("missing", "missing", Item{}).
            WithError(testx.ErrorIs(ErrNotFound)),
    )
}
```

Use a hand-written test instead when the subject needs a deadline, request-scoped values, or a specifically cancelled context; the preset always creates a fresh background context.

## Interface contract

```go
type Store interface {
    Set(string, string)
    Get(string) (string, bool)
}

var StoreContract = testx.NewContract("Store",
    testx.S("round trip", func(t *testing.T, store Store) {
        store.Set("key", "value")
        got, ok := store.Get("key")
        testx.Assert(t, ok).True()
        testx.Assert(t, got).Equal("value")
    }),
)

func TestStoreImplementations(t *testing.T) {
    StoreContract.VerifyAll(t,
        testx.Impl("memory", func(t *testing.T) Store {
            return NewMemoryStore()
        }),
    )
}
```

Generate contracts for observable interface behavior, not implementation details. Factories should return fresh isolated implementations and may use `t.TempDir()` or `t.Cleanup()`.

## HTTP handler

```go
func TestHandler(t *testing.T) {
    testx.HTTP(handler).Run(t,
        testx.C("create",
            testx.HTTPRequest{
                Method: http.MethodPost,
                Target: "/items",
                Body:   []byte(`{"name":"one"}`),
            },
            testx.HTTPResponse{
                Status: http.StatusCreated,
                Body:   `{"id":"1"}`,
            },
        ),
    )
}
```

Only expected headers are compared. Use `JSONEqual` in a hand-written test when response JSON formatting is not contractual, because `HTTPResponse.Body` comparison is exact.

## Eventually obtain a value

```go
func TestWorker(t *testing.T) {
    value, ok := testx.EventuallyValue(t, func() (string, bool) {
        return store.Lookup("job")
    }).Every(5 * time.Millisecond).Within(time.Second)
    if !ok {
        return // failure was already reported
    }
    testx.Assert(t, value).Equal("done")
}
```

Do not use polling to hide a deterministic synchronization bug. Keep timeouts bounded.

## Golden text

```go
func TestRender(t *testing.T) {
    testx.GoldenString(t, "testdata/render.golden", Render(input))
}
```

Update intentionally with `TESTX_UPDATE_GOLDEN=1 go test ./...`, inspect the diff, and commit the fixture. Do not set the environment variable from the test.

## Benchmark and fuzzing

```go
func BenchmarkParse(b *testing.B) {
    testx.Benchmark(b, func(input string) {
        _, _ = Parse(input)
    },
        testx.B("short", "INFO ready"),
        testx.B("long", strings.Repeat("x", 1024)),
    )
}

func FuzzParse(f *testing.F) {
    testx.FuzzSeed(f, "INFO ready")
    testx.FuzzSeed(f, "")
    f.Fuzz(func(t *testing.T, input string) {
        _, _ = Parse(input)
    })
}
```
