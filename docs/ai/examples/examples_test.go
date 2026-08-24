package examples_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/go-testx/testx"
)

type entry struct {
	Level   string
	Message string
}

func parseLine(raw string) (entry, error) {
	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 {
		return entry{}, errors.New("invalid log line")
	}
	if parts[0] == "BAD" {
		return entry{}, errors.New("invalid level")
	}
	return entry{Level: parts[0], Message: parts[1]}, nil
}

func TestRunErrRecipe(t *testing.T) {
	testx.RunErr(t, parseLine,
		testx.C("valid info", "INFO ready", entry{Level: "INFO", Message: "ready"}),
		testx.C("invalid level", "BAD ready", entry{}).
			WithError(testx.ErrorContains("invalid level")),
	)
}

func TestSelectedFieldsRecipe(t *testing.T) {
	got, err := parseLine("INFO ready")
	testx.Require(t, err).NoError()
	testx.Assert(t, got.Level).Equal("INFO")
	testx.Assert(t, got.Message).Equal("ready")
}

func TestBoundAssertionsRecipe(t *testing.T) {
	got, err := parseLine("DEBUG connected")
	check := testx.New(t)
	check.Require(err).NoError()
	check.Assert(got.Level).Equal("DEBUG")
	check.Assert(got.Message).Contains("connect")
}
