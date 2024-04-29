package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func funcNames(filename string, exportOnly bool) []string {
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, filename, nil, 0)
	if exportOnly {
		ast.FileExports(file) // trim AST
	}

	funcNames := []string{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl) // assert type of declaration
		if !ok {
			continue
		}
		funcNames = append(funcNames, fn.Name.Name)
	}
	return funcNames
}

func GetBenchmarks(root string) ([]string, error) {
	var data []string
	fileSystem := os.DirFS(root)
	err := fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		if !strings.HasSuffix(filepath.Base(path), "_test.go") {
			return nil
		}
		fnames := funcNames(path, false)
		for _, fname := range fnames {
			if strings.HasPrefix(fname, "Benchmark") {
				data = append(data, fname)
			}
		}
		return nil
	})
	return data, err
}
