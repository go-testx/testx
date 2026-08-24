package testx

import "testing"

// TestT is intentionally *testing.T rather than a custom test context.
// testx orchestrators create real Go subtests and remain compatible with go test.
type TestT = *testing.T
