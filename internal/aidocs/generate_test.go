package main

import (
	"bytes"
	"os"
	"testing"
)

func TestGeneratedDocumentationIsCurrent(t *testing.T) {
	root, err := findRoot()
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := render(root)
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range outputs {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read generated file %s: %v", path, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("generated AI documentation is stale: %s; run go generate ./internal/aidocs", path)
		}
	}
}
