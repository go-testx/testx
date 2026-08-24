# testx API Reference for Code Generation

This reference describes the public API of `github.com/go-testx/testx`. Use the exact names and signatures shown here.

## Testing type

```go
type TestT = *testing.T
```

`TestT` is an alias, not a custom test context. APIs that need subtests use the real `*testing.T`, so standard methods such as `Helper`, `Run`, `Cleanup`, and `TempDir` remain available.

## Assertions

```go
func New(t testing.TB) Assertions
func Assert[T any](t testing.TB, actual T) Assertion[T]
func Require[T any](t testing.TB, actual T) Assertion[T]

func (Assertions) Assert(actual any) Assertion[any]
func (Assertions) Require(actual any) Assertion[any]
```

`Assert` reports failures with `Errorf` and lets the test continue. `Require` reports with `Fatalf` and stops the current test goroutine. Assertion methods return `bool` for optional control flow, but the failure is already reported.

Available fluent methods:

```text
Because(format, args...)
Equal(expected, cmpOptions...)
NotEqual(unexpected, cmpOptions...)
Nil() / NotNil()
True() / False()
Len(expected)
Empty()
Contains(expected) / NotContains(expected)
ContainsAll(expected) / ElementsMatch(expected)
MatchRegexp(pattern)
NoError() / Error()
ErrorIs(target) / ErrorAs(target) / ErrorContains(substring)
JSONEqual(expected, cmpOptions...)
Panics() / PanicsWith(expected) / NotPanics()
```

`Contains` supports string substrings, array/slice members, and map keys. `ContainsAll` and `ElementsMatch` honor duplicate elements. Panic assertions require a non-nil, zero-argument function.

There is also a non-fatal package helper:

```go
func JSONEqual(t testing.TB, actual, expected any, options ...cmp.Option) bool
```

## Cases and runners

```go
type Case[I, O any] struct {
    Name   string
    Input  I
    Expect O
}

func C[I, O any](name string, input I, expect O) Case[I, O]
func Cases[I, O any](cases ...Case[I, O]) CaseSet[I, O]

func (Case[I, O]) WithCmp(options ...cmp.Option) Case[I, O]
func (Case[I, O]) WithError(expect ErrorExpectation) Case[I, O]
func (Case[I, O]) CompareOutputOnError() Case[I, O]
func (Case[I, O]) Parallel() Case[I, O]
func (Case[I, O]) Skip(reason string) Case[I, O]

func Run[I, O any](t *testing.T, fn func(I) O, cases ...Case[I, O])
func RunErr[I, O any](t *testing.T, fn func(I) (O, error), cases ...Case[I, O])
func (CaseSet[I, O]) Run(t *testing.T, fn func(I) O)
func (CaseSet[I, O]) RunErr(t *testing.T, fn func(I) (O, error))
```

`Run` and `RunErr` create standard `t.Run` subtests. An empty Case name becomes `"case"`.

Reusable presets:

```go
func Func[I, O any](fn func(I) O) FuncPreset[I, O]
func (FuncPreset[I, O]) Run(t *testing.T, cases ...Case[I, O])

func FuncErr[I, O any](fn func(I) (O, error)) FuncErrPreset[I, O]
func (FuncErrPreset[I, O]) Run(t *testing.T, cases ...Case[I, O])

func ContextFuncErr[I, O any](fn func(context.Context, I) (O, error)) ContextFuncErrPreset[I, O]
func (ContextFuncErrPreset[I, O]) Run(t *testing.T, cases ...Case[I, O])
```

`ContextFuncErr` creates a fresh cancellable background context for each Case.

## Error expectations

The zero value of `ErrorExpectation` means no error is expected.

```go
func AnyError() ErrorExpectation
func ErrorIs(target error) ErrorExpectation
func ErrorContains(substring string) ErrorExpectation
func ErrorMatch(description string, predicate func(error) bool) ErrorExpectation
```

Attach an expectation to a Case:

```go
testx.C("missing", input, Output{}).
    WithError(testx.ErrorIs(ErrNotFound))
```

When an expected error matches, `RunErr`, `FuncErr`, and `ContextFuncErr` normally return without comparing `Expect`. Chain `CompareOutputOnError()` to compare it.

## Contracts

```go
type Factory[T any] func(*testing.T) T
type Spec[T any] struct { Name string; Test func(*testing.T, T) }
type Implementation[T any] struct { Name string; Factory Factory[T] }

func S[T any](name string, test func(*testing.T, T)) Spec[T]
func NewContract[T any](name string, specs ...Spec[T]) Contract[T]
func Impl[T any](name string, factory Factory[T]) Implementation[T]

func (Contract[T]) Verify(t *testing.T, factory Factory[T])
func (Contract[T]) VerifyAs(t *testing.T, name string, factory Factory[T])
func (Contract[T]) VerifyAll(t *testing.T, implementations ...Implementation[T])
```

Each spec receives a fresh value from the factory. Use `VerifyAll` when the same interface contract must run against several implementations.

## Eventually

```go
func Eventually(t testing.TB, condition func() bool) EventuallyAssertion
func (EventuallyAssertion) Every(interval time.Duration) EventuallyAssertion
func (EventuallyAssertion) Within(timeout time.Duration) bool

func EventuallyValue[T any](t testing.TB, condition func() (T, bool)) EventuallyValueAssertion[T]
func (EventuallyValueAssertion[T]) Every(interval time.Duration) EventuallyValueAssertion[T]
func (EventuallyValueAssertion[T]) Within(timeout time.Duration) (T, bool)
```

The default poll interval is 10 ms. Both interval and timeout must be positive. `EventuallyValue` returns the ready value and a boolean.

## HTTP

```go
type HTTPRequest struct {
    Method string
    Target string
    Body   []byte
    Header http.Header
}

type HTTPResponse struct {
    Status int
    Body   string
    Header http.Header
}

func HTTP(handler http.Handler) HTTPPreset
func (HTTPPreset) Run(t *testing.T, cases ...Case[HTTPRequest, HTTPResponse])
```

An empty request method defaults to `GET`, an empty target to `/`, and expected status zero to `200`. Expected response headers are a subset; only listed headers are compared. Body comparison is exact.

## CLI

```go
type CLIRequest struct {
    Path    string
    Args    []string
    Dir     string
    Env     []string
    Stdin   string
    Timeout time.Duration
}

type CLIResult struct {
    ExitCode int
    Stdout   string
    Stderr   string
    TimedOut bool
}

func CLI() CLIPreset
func (CLIPreset) Run(t *testing.T, cases ...Case[CLIRequest, CLIResult])
```

The preset uses `exec.CommandContext` directly and never invokes a shell. Non-zero exit codes and timeouts are expected result data, not runner errors.

## Golden files and snapshots

```go
func Golden(t testing.TB, path string, actual []byte) bool
func GoldenString(t testing.TB, path string, actual string) bool
func Snapshot(t testing.TB, name string, actual []byte) bool
func SnapshotString(t testing.TB, name string, actual string) bool
```

Comparison is read-only by default. Set `TESTX_UPDATE_GOLDEN=1` to update. Snapshots are stored below `testdata/snapshots`. Never enable updates unconditionally inside test code.

## Benchmarks and fuzz seeds

```go
func B[I any](name string, input I) BenchmarkCase[I]
func Benchmark[I any](b *testing.B, fn func(I), cases ...BenchmarkCase[I])
func FuzzSeed(f *testing.F, values ...any)
func FuzzSeeds(f *testing.F, seeds ...[]any)
```

`Benchmark` creates named sub-benchmarks and reports allocations. Fuzz helpers add seed corpora; the actual fuzz function remains standard `f.Fuzz(...)`.
