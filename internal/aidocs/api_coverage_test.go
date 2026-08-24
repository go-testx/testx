package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPIReferenceMentionsEveryExport(t *testing.T) {
	root, err := findRoot()
	if err != nil {
		t.Fatal(err)
	}
	reference, err := os.ReadFile(filepath.Join(root, "docs", "ai", "api-reference.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(reference)
	fset := token.NewFileSet()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(root, name), nil, 0)
		if err != nil {
			t.Errorf("parse %s: %v", name, err)
			continue
		}
		for _, declaration := range file.Decls {
			switch node := declaration.(type) {
			case *ast.FuncDecl:
				if node.Name.IsExported() && !strings.Contains(text, node.Name.Name) {
					t.Errorf("docs/ai/api-reference.md does not mention exported function or method %s", node.Name.Name)
				}
			case *ast.GenDecl:
				for _, spec := range node.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if ok && typeSpec.Name.IsExported() && !strings.Contains(text, typeSpec.Name.Name) {
						t.Errorf("docs/ai/api-reference.md does not mention exported type %s", typeSpec.Name.Name)
					}
				}
			}
		}
	}
}
