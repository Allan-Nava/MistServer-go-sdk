package mist_go

import (
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

// TestExamplesAreDocumented fails for any Example that go/doc (and so
// pkg.go.dev) can't attach to a documented identifier. Examples named after
// interface methods or variables still run, but never show up in the docs.
func TestExamplesAreDocumented(t *testing.T) {
	fset := token.NewFileSet()
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var files []*ast.File
	for _, path := range paths {
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}

	pkg, err := doc.NewFromFiles(fset, files, "github.com/Allan-Nava/MistServer-go-sdk/mist")
	if err != nil {
		t.Fatal(err)
	}

	shown := map[string]bool{}
	add := func(examples []*doc.Example) {
		for _, e := range examples {
			shown[e.Name] = true
		}
	}
	add(pkg.Examples)
	for _, f := range pkg.Funcs {
		add(f.Examples)
	}
	for _, typ := range pkg.Types {
		add(typ.Examples)
		for _, f := range typ.Funcs {
			add(f.Examples)
		}
		for _, m := range typ.Methods {
			add(m.Examples)
		}
	}

	for _, e := range doc.Examples(files...) {
		if !shown[e.Name] {
			t.Errorf("Example%s is not attached to any documented identifier; pkg.go.dev won't show it", e.Name)
		}
	}
}
