# testx Generation Constraints and Review Checklist

Apply these constraints when an LLM creates, reviews, or modifies Go tests that use testx.

## Do not generate these patterns

### Duplicate failure reporting

Incorrect:

```go
if !testx.Assert(t, got).Equal(want) {
    t.Fatal("values differ")
}
```

Correct:

```go
testx.Assert(t, got).Equal(want)
```

Assertion methods already call `Errorf` or `Fatalf`. Use their boolean result only for control flow that prevents unsafe follow-up work, never to report the same failure again.

### Using a result after a failed non-fatal error check

Risky:

```go
got, err := Load()
testx.Assert(t, err).NoError()
testx.Assert(t, got.Name).Equal("item")
```

Prefer:

```go
got, err := Load()
testx.Require(t, err).NoError()
testx.Assert(t, got.Name).Equal("item")
```

### Adding `.WithError(...)` to a normal Case

`RunErr` expects `nil` by default. Do not add `.WithError(testx.AnyError())` unless that exact Case should fail.

### Forcing unsupported function signatures into runners

`Run` accepts exactly `func(I) O`. `RunErr` accepts exactly `func(I) (O, error)`. `ContextFuncErr` accepts exactly `func(context.Context, I) (O, error)`. For multiple independent inputs, multiple outputs, variadic arguments, injected context values, or complex setup, write an adapter closure or use a normal test.

### Comparing unstable or irrelevant fields

Avoid whole-struct equality for timestamps, generated IDs, random data, caches, mutexes, or unexported implementation state unless those fields are contractual. Compare selected fields or supply deliberate `cmp.Option` values.

### Updating golden files implicitly

Never call `os.Setenv("TESTX_UPDATE_GOLDEN", "1")` from a test. Golden updates must be an explicit developer action so CI remains read-only.

### Guessing fixtures and business behavior

Do not present zero values or invented examples as authoritative expected behavior. Infer cases from implementation, public documentation, existing tests, or user requirements. Mark unresolved expectations clearly and ask for domain input when required.

## Selection heuristics

- Prefer `Require` for prerequisites: setup errors, decoding errors, missing values, and conditions that make later access unsafe.
- Prefer `Assert` for independent checks so one test can report several useful differences.
- Prefer `ErrorIs` over error text when a sentinel or typed wrapping contract exists.
- Prefer explicit field assertions when only a few fields matter.
- Prefer `Run`/`RunErr` for pure single-input/single-output subjects with repetitive cases.
- Prefer standard `t.Run` when Cases need different mocks, setup, call sequences, or assertions.
- Prefer `Contract` for behavior shared by multiple interface implementations.
- Preserve existing package style (`package foo` versus `package foo_test`) unless there is a reason to change it.
- Keep generated tests deterministic, isolated, bounded, and safe for `go test ./...`.

## Review checklist

Before returning generated test code:

1. Verify every testx identifier exists in `api-reference.md`.
2. Verify the runner matches the subject function signature.
3. Verify normal `RunErr` Cases expect no error and error Cases use the right expectation.
4. Verify fatal prerequisites use `Require`.
5. Verify assertions are not followed by duplicate `Fatal`, `Error`, or `Log` reporting.
6. Verify Case names explain behavior and do not collide.
7. Verify expected values come from known behavior rather than placeholder assumptions.
8. Verify shared mutable fixtures are not used by parallel Cases.
9. Verify timeouts and external processes are bounded.
10. Format the file and run the narrow test, then `go test ./...` when available.

## GoLand static-discovery notes

The optional GoLand plugin can run static Cases directly. Its precise Case gutter support works best with:

- the exact module import `github.com/go-testx/testx`, including an explicit alias;
- string literals, compile-time constants, and simple constant concatenation for names;
- inline Cases or a same-file package CaseSet used by one `TestXxx`.

Dynamic names, dot imports, arbitrary wrapper functions, cross-file CaseSets, and one CaseSet shared by several tests may run correctly but do not necessarily receive a precise Case gutter action.
