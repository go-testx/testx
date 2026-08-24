package testx

import "testing"

// BenchmarkCase describes one named benchmark input.
type BenchmarkCase[I any] struct {
	Name  string
	Input I
}

// B creates a benchmark case with type inference.
func B[I any](name string, input I) BenchmarkCase[I] {
	return BenchmarkCase[I]{Name: name, Input: input}
}

// Benchmark runs a subject for each named input and reports allocations.
func Benchmark[I any](b *testing.B, fn func(I), cases ...BenchmarkCase[I]) {
	b.Helper()
	if fn == nil {
		b.Fatal("testx: benchmark subject function is nil")
	}
	for _, c := range cases {
		c := c
		name := c.Name
		if name == "" {
			name = "case"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				fn(c.Input)
			}
		})
	}
}

// FuzzSeeds adds seed corpora to a fuzz test. Each inner slice is passed to testing.F.Add.
func FuzzSeeds(f *testing.F, seeds ...[]any) {
	f.Helper()
	for i, seed := range seeds {
		if len(seed) == 0 {
			f.Fatalf("testx: fuzz seed %d is empty", i)
		}
		f.Add(seed...)
	}
}

// FuzzSeed adds one seed corpus to a fuzz test.
func FuzzSeed(f *testing.F, values ...any) {
	f.Helper()
	if len(values) == 0 {
		f.Fatal("testx: fuzz seed is empty")
	}
	f.Add(values...)
}
