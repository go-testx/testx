package testx_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/go-testx/testx"
)

type parseResult struct {
	Level   string
	Message string
}

func parseLine(s string) (parseResult, error) {
	parts := strings.SplitN(s, " ", 2)
	if len(parts) == 0 || parts[0] == "" {
		return parseResult{}, errors.New("empty input")
	}
	if parts[0] == "BAD" {
		return parseResult{}, fmt.Errorf("invalid level: %s", parts[0])
	}
	message := ""
	if len(parts) == 2 {
		message = parts[1]
	}
	return parseResult{Level: parts[0], Message: message}, nil
}

func TestLevel2Primitives(t *testing.T) {
	cases := []testx.Case[string, parseResult]{
		testx.C("info", "INFO hello", parseResult{Level: "INFO", Message: "hello"}),
		testx.C("debug", "DEBUG access", parseResult{Level: "DEBUG", Message: "access"}),
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			got, err := parseLine(c.Input)
			testx.Require(t, err).NoError()
			testx.Assert(t, got).Equal(c.Expect)
			testx.Assert(t, got.Message).Contains("e")
		})
	}
}

func TestRunErr(t *testing.T) {
	testx.RunErr(t, parseLine,
		testx.C("info", "INFO hello", parseResult{Level: "INFO", Message: "hello"}),
		testx.C("bad", "BAD hello", parseResult{}).WithError(testx.ErrorContains("invalid level")),
		testx.C("bad partial", "BAD hello", parseResult{}).
			WithError(testx.ErrorMatch("invalid error", func(err error) bool {
				return strings.Contains(err.Error(), "invalid")
			})).
			CompareOutputOnError(),
	)
}

func TestCaseSetAndPreset(t *testing.T) {
	cases := testx.Cases(
		testx.C("one", " INFO ", "INFO"),
		testx.C("two", " DEBUG ", "DEBUG"),
	)

	cases.Run(t, strings.TrimSpace)
	testx.Func(strings.ToLower).Run(t,
		testx.C("lower", "HELLO", "hello"),
	)
}

func TestCaseCmpOptions(t *testing.T) {
	type result struct {
		Items []string
	}

	testx.Run(t,
		func(_ string) result { return result{Items: nil} },
		testx.C("cmp option", "x", result{Items: []string{}}).WithCmp(cmpopts.EquateEmpty()),
	)
}

type store interface {
	Set(string, string)
	Get(string) (string, bool)
}

type memoryStore map[string]string

func (m memoryStore) Set(k, v string) { m[k] = v }
func (m memoryStore) Get(k string) (string, bool) {
	v, ok := m[k]
	return v, ok
}

func storeContract() testx.Contract[store] {
	return testx.NewContract("store",
		testx.S("set then get", func(t *testing.T, s store) {
			s.Set("foo", "bar")
			v, ok := s.Get("foo")
			testx.Assert(t, ok).True()
			testx.Assert(t, v).Equal("bar")
		}),
		testx.S("fresh instance per spec", func(t *testing.T, s store) {
			_, ok := s.Get("foo")
			testx.Assert(t, ok).False()
		}),
	)
}

func TestContract(t *testing.T) {
	contract := storeContract()
	contract.VerifyAll(t,
		testx.Impl("memory-a", func(t *testing.T) store {
			return memoryStore{}
		}),
		testx.Impl("memory-b", func(t *testing.T) store {
			return memoryStore{}
		}),
	)
}

func TestEventually(t *testing.T) {
	var ready atomic.Bool
	go func() {
		time.Sleep(20 * time.Millisecond)
		ready.Store(true)
	}()

	testx.Eventually(t, ready.Load).Every(5 * time.Millisecond).Within(time.Second)
}

func TestCaseSkipWithEmptyReason(t *testing.T) {
	called := atomic.Bool{}
	testx.Run(t,
		func(input string) string {
			called.Store(true)
			return input
		},
		testx.C("empty reason", "value", "value").Skip(""),
	)
	if called.Load() {
		t.Fatal("skipped case executed")
	}
}

func TestEventuallyDoesNotSleepPastDeadline(t *testing.T) {
	if os.Getenv("TESTX_EVENTUALLY_CHILD") == "1" {
		testx.Eventually(t, func() bool { return false }).Every(5 * time.Second).Within(15 * time.Millisecond)
		return
	}
	start := time.Now()
	cmd := exec.Command(os.Args[0], "-test.run=^TestEventuallyDoesNotSleepPastDeadline$")
	cmd.Env = append(os.Environ(), "TESTX_EVENTUALLY_CHILD=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("eventually timeout unexpectedly passed")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("eventually exceeded bounded timeout: %s", elapsed)
	}
}

func TestErrorAsInvalidTargetReportsFailure(t *testing.T) {
	if os.Getenv("TESTX_ERROR_AS_CHILD") == "1" {
		testx.Assert(t, errors.New("boom")).ErrorAs(nil)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestErrorAsInvalidTargetReportsFailure$")
	cmd.Env = append(os.Environ(), "TESTX_ERROR_AS_CHILD=1")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("invalid ErrorAs target unexpectedly passed")
	}
	if _, ok := err.(*exec.ExitError); !ok {
		t.Fatalf("expected test process failure, got %T: %v", err, err)
	}
	if strings.Contains(string(output), "panic:") {
		t.Fatalf("ErrorAs panicked instead of reporting a failure:\n%s", output)
	}
	if !strings.Contains(string(output), "could not inspect target") {
		t.Fatalf("missing ErrorAs diagnostic:\n%s", output)
	}
}

func TestBoundAssertions(t *testing.T) {
	check := testx.New(t)
	check.Assert([]int{1, 2, 2, 3}).ContainsAll([]int{2, 2})
	check.Assert("INFO hello").MatchRegexp(`^INFO`)
	check.Require(errors.New("boom")).Error()
}

func TestAssertionsSmoke(t *testing.T) {
	check := testx.New(t)
	check.Assert(1).NotEqual(2)
	var ptr *string
	check.Assert(ptr).Nil()
	value := "value"
	check.Assert(&value).NotNil()
	check.Assert(value).Len(5)
	check.Assert([]int{1, 2}).Contains(2)
	check.Assert("INFO hello").MatchRegexp(`^INFO`)
	check.Assert([]int{}).Empty()
	check.Assert(0).Empty()
	check.Assert([2]int{}).Because("zero array").Empty()
	var target *os.PathError
	check.Assert(error(&os.PathError{Op: "open"})).ErrorAs(&target)
	if target == nil {
		t.Fatal("ErrorAs should populate target")
	}
	if target.Op != "open" {
		t.Fatalf("unexpected target: %#v", target)
	}
	sentinel := errors.New("sentinel")
	check.Assert(error(sentinel)).Error()
	check.Assert(error(sentinel)).ErrorIs(sentinel)
	check.Assert(error(sentinel)).ErrorContains("tine")
}

func TestAdditionalRunners(t *testing.T) {
	cases := testx.Cases(testx.C("case set error", "x", "x"))
	cases.RunErr(t, func(value string) (string, error) { return value, nil })
	testx.FuncErr(func(value string) (string, error) { return value, nil }).Run(t,
		testx.C("preset error", "x", "x"),
	)
	testx.Run(t, strings.ToUpper,
		testx.C("parallel", "x", "X").Parallel(),
	)
	contract := storeContract()
	contract.Verify(t, func(t *testing.T) store { return memoryStore{} })
	contract.VerifyAs(t, "named", func(t *testing.T) store { return memoryStore{} })
}
