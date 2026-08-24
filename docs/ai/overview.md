# testx AI Guide

`github.com/go-testx/testx` is a Go 1.23+ testing toolkit layered on the standard `testing` package. It does not replace `testing.T` or the Go test runner. Use standard Go test functions and add only the testx abstraction that makes a test clearer.

## Install and import

```bash
go get github.com/go-testx/testx
```

```go
import "github.com/go-testx/testx"
```

## Choose an abstraction level

| Need | Prefer |
| --- | --- |
| Full control over setup, cleanup, loops, and subtests | Standard `testing` plus `testx.Assert` / `testx.Require` |
| Typed table data with hand-written `t.Run` | `testx.Case[I, O]` or `testx.C(...)` |
| Test `func(I) O` | `testx.Run` or `testx.Func(fn).Run` |
| Test `func(I) (O, error)` | `testx.RunErr` or `testx.FuncErr(fn).Run` |
| Test `func(context.Context, I) (O, error)` | `testx.ContextFuncErr(fn).Run` |
| Verify several implementations of one interface | `testx.Contract` |
| Poll asynchronous state | `testx.Eventually` or `testx.EventuallyValue` |
| Test an `http.Handler` | `testx.HTTP(handler).Run` |
| Test a process without invoking a shell | `testx.CLI().Run` |
| Compare semantic JSON | `JSONEqual` |
| Compare files or snapshots | `Golden`, `GoldenString`, `Snapshot`, `SnapshotString` |

Start with standard `testing` and assertions when the flow is conditional or has substantial setup. Use declarative runners only when the subject has one of their exact supported signatures.

## Core rules for generated code

1. Keep standard signatures such as `func TestName(t *testing.T)`.
2. Treat `Assert` as non-fatal and `Require` as fatal. Both report failures themselves.
3. Never wrap an assertion in `if !... { t.Fatal(...) }` or add a second `t.Error`; that reports the same failure twice.
4. Check an error with `Require(...).NoError()` before using a result that is invalid on error.
5. `RunErr` expects no error by default. Add `.WithError(...)` only to cases that should return an error.
6. Expected-error cases skip output comparison by default. Add `.CompareOutputOnError()` only when the partial output is part of the contract.
7. `Equal` and declarative case comparison use `go-cmp`. Pass explicit `cmp.Option` values for unexported fields or domain-specific equality.
8. Prefer stable, descriptive literal Case names. They produce ordinary `t.Run` subtests and work best with IDE tooling.
9. Do not invent testx APIs. Fall back to `testing.T` when the API reference has no matching operation.
10. Run `go test ./...` after generating or modifying tests.

## Minimal examples

For a function returning a value and an error:

```go
func TestAtoi(t *testing.T) {
    testx.RunErr(t, strconv.Atoi,
        testx.C("decimal", "42", 42),
        testx.C("invalid", "not-a-number", 0).
            WithError(testx.AnyError()),
    )
}
```

For custom flow and field-only comparison:

```go
func TestParse(t *testing.T) {
    got, err := parser.Parse(raw)
    testx.Require(t, err).NoError()
    testx.Assert(t, got.Level).Equal(want.Level)
    testx.Assert(t, got.Message).Equal(want.Message)
}
```

Bind the test once when many assertions are needed:

```go
func TestResult(t *testing.T) {
    check := testx.New(t)
    check.Require(err).NoError()
    check.Assert(got.Level).Equal("INFO")
    check.Assert(got.Message).Contains("ready")
}
```

Package-level `Assert` preserves compile-time matching between actual and expected generic types. Bound `check.Assert` accepts `any` for convenience.

## Related reference

- Read `api-reference.md` for exact exported signatures and semantics.
- Read `recipes.md` for complete patterns by test type.
- Read `constraints.md` before generating or rewriting tests.
