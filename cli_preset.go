package testx

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

// CLIRequest describes a command execution without invoking a shell.
type CLIRequest struct {
	Path    string
	Args    []string
	Dir     string
	Env     []string
	Stdin   string
	Timeout time.Duration
}

// CLIResult captures a completed process.
type CLIResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	TimedOut bool
}

// CLIPreset executes subprocess cases.
type CLIPreset struct{}

// CLI creates a command preset.
func CLI() CLIPreset { return CLIPreset{} }

// Run executes CLI cases. Non-zero exit codes are captured in CLIResult rather than treated as runner errors.
func (CLIPreset) Run(t *testing.T, cases ...Case[CLIRequest, CLIResult]) {
	t.Helper()
	for i := range cases {
		c := cases[i]
		name := c.Name
		if name == "" {
			name = "case"
		}
		t.Run(name, func(t *testing.T) {
			if c.skip {
				t.Skip(c.skipReason)
			}
			if c.parallel {
				t.Parallel()
			}
			got, err := runCLI(t, c.Input)
			if err != nil {
				t.Fatalf("testx.CLI: %v", err)
			}
			assertCaseEqual(t, c.Expect, got, c.cmpOptions)
		})
	}
}

func runCLI(t *testing.T, request CLIRequest) (CLIResult, error) {
	t.Helper()
	if request.Path == "" {
		return CLIResult{}, errors.New("command path is empty")
	}
	ctx := context.Background()
	cancel := func() {}
	if request.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, request.Timeout)
	}
	defer cancel()
	cmd := exec.CommandContext(ctx, request.Path, request.Args...)
	cmd.Dir = request.Dir
	cmd.Env = append(os.Environ(), request.Env...)
	cmd.Stdin = bytes.NewBufferString(request.Stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := CLIResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if err == nil {
		return result, nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		result.TimedOut = true
		return result, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}
	return result, err
}
