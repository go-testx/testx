package testx_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/go-testx/testx"
)

func TestContextFuncErr(t *testing.T) {
	testx.ContextFuncErr(func(ctx context.Context, input string) (string, error) {
		if ctx == nil {
			return "", errors.New("nil context")
		}
		return strings.ToUpper(input), nil
	}).Run(t,
		testx.C("upper", "hello", "HELLO"),
	)
}

func TestHTTPPreset(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Testx", r.Header.Get("X-Input"))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(r.Method + " " + r.URL.Path))
	})
	testx.HTTP(handler).Run(t,
		testx.C("create",
			testx.HTTPRequest{
				Method: http.MethodPost,
				Target: "/items",
				Header: http.Header{"X-Input": []string{"ok"}},
			},
			testx.HTTPResponse{
				Status: http.StatusCreated,
				Body:   "POST /items",
				Header: http.Header{"X-Testx": []string{"ok"}},
			},
		),
	)
}

func TestJSONEqual(t *testing.T) {
	actual := `{"count": 1, "items": ["a", "b"]}`
	expected := `{
  "items": ["a", "b"],
  "count": 1.0
}`
	check := testx.New(t)
	check.Assert(actual).JSONEqual(expected)
	testx.JSONEqual(t, []byte(`{"ok":true}`), map[string]bool{"ok": true})
}

func TestGoldenAndSnapshot(t *testing.T) {
	testx.GoldenString(t, filepath.Join("testdata", "golden", "hello.golden"), "hello\n")
}

func TestSnapshotFixture(t *testing.T) {
	testx.SnapshotString(t, "basic", "snapshot\n")
}

func TestGoldenUpdate(t *testing.T) {
	t.Setenv("TESTX_UPDATE_GOLDEN", "1")
	path := filepath.Join(t.TempDir(), "nested", "value.golden")
	testx.GoldenString(t, path, "updated\n")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "updated\n" {
		t.Fatalf("unexpected golden content: %q", got)
	}
}

func TestPanicAssertions(t *testing.T) {
	check := testx.New(t)
	check.Assert(func() { panic("boom") }).Panics()
	check.Assert(func() { panic("boom") }).PanicsWith("boom")
	check.Assert(func() {}).NotPanics()
}

func TestEventuallyValue(t *testing.T) {
	var attempts atomic.Int32
	value, ok := testx.EventuallyValue(t, func() (string, bool) {
		if attempts.Add(1) < 3 {
			return "", false
		}
		return "ready", true
	}).Every(time.Millisecond).Within(time.Second)
	if !ok || value != "ready" {
		t.Fatalf("unexpected eventual value: %q, %v", value, ok)
	}
}

func TestCollectionMatchers(t *testing.T) {
	check := testx.New(t)
	check.Assert([]int{1, 2, 2, 3}).ContainsAll([]int{2, 2})
	check.Assert([]int{1, 2, 2}).ElementsMatch([]int{2, 1, 2})
	check.Assert([]string{"a", "b"}).NotContains("c")
	check.Assert(map[string]int{"key": 1}).Contains("key")
}

func TestCLIPreset(t *testing.T) {
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go command not found")
	}
	testx.CLI().Run(t,
		testx.C("go version",
			testx.CLIRequest{Path: goCommand, Args: []string{"env", "GOVERSION"}},
			testx.CLIResult{Stdout: runtime.Version() + "\n"},
		),
	)
}

func TestCLITimeout(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	testx.CLI().Run(t,
		testx.C("timeout",
			testx.CLIRequest{
				Path:    executable,
				Args:    []string{"-test.run=^TestCLISleepHelper$"},
				Env:     []string{"TESTX_CLI_SLEEP=1"},
				Timeout: 20 * time.Millisecond,
			},
			testx.CLIResult{TimedOut: true},
		).WithCmp(cmpopts.IgnoreFields(testx.CLIResult{}, "ExitCode", "Stdout", "Stderr")),
	)
}

func TestCLISleepHelper(t *testing.T) {
	if os.Getenv("TESTX_CLI_SLEEP") != "1" {
		return
	}
	time.Sleep(time.Second)
}

func BenchmarkHelper(b *testing.B) {
	testx.Benchmark(b, func(value string) {
		_ = strings.ToUpper(value)
	},
		testx.B("short", "hello"),
		testx.B("long", strings.Repeat("x", 64)),
	)
}

func FuzzSeedHelper(f *testing.F) {
	testx.FuzzSeeds(f,
		[]any{"hello"},
		[]any{""},
	)
	testx.FuzzSeed(f, "world")
	f.Fuzz(func(t *testing.T, value string) {
		if len(strings.ToUpper(value)) != len(value) {
			t.Skip("Unicode uppercasing may change byte length")
		}
	})
}
