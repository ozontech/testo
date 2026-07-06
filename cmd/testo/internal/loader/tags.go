package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func BuildTags(testOnly bool) (add, remove map[string]struct{}, err error) {
	add = make(map[string]struct{})
	remove = make(map[string]struct{})

	files, err := listGoFiles(context.Background(), testOnly)
	if err != nil {
		return nil, nil, err
	}

	for _, f := range files {
		buildTags(f, add, remove)
	}

	return add, remove, nil
}

func buildTags(file *ast.File, add, remove map[string]struct{}) {
	for _, g := range file.Comments {
		for _, c := range g.List {
			if !constraint.IsGoBuild(c.Text) && !constraint.IsPlusBuild(c.Text) {
				continue
			}

			expr, err := constraint.Parse(c.Text)
			if err != nil {
				continue
			}

			addConstraintExpr(expr, add, remove)

		}
	}
}

func addConstraintExpr(e constraint.Expr, add, remove map[string]struct{}) {
	switch e := e.(type) {
	case *constraint.AndExpr:
		addConstraintExpr(e.X, add, remove)
		addConstraintExpr(e.Y, add, remove)

	case *constraint.NotExpr:
		addConstraintExpr(e.X, remove, add)

	case *constraint.OrExpr:
		addConstraintExpr(e.X, add, remove)
		addConstraintExpr(e.Y, add, remove)

	case *constraint.TagExpr:
		add[e.Tag] = struct{}{}
	}
}

func listGoFiles(ctx context.Context, testOnly bool) ([]*ast.File, error) {
	g := exec.CommandContext(ctx, "go", "list", "-m", "-json")

	out, err := g.Output()
	if err != nil {
		return nil, err
	}

	var mod struct {
		Dir string
	}

	err = json.Unmarshal(out, &mod)
	if err != nil {
		return nil, err
	}

	if mod.Dir == "" {
		return nil, fmt.Errorf("outside of go module")
	}

	files := make(map[string]struct{})

	fs.WalkDir(os.DirFS(mod.Dir), ".", func(path string, d fs.DirEntry, err error) error {
		if path == "testdata" && d.IsDir() {
			return fs.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		suffix := ".go"
		if testOnly {
			suffix = "_test.go"
		}

		if !strings.HasSuffix(d.Name(), suffix) {
			return nil
		}

		files[filepath.Join(mod.Dir, path)] = struct{}{}

		return nil
	})

	s := make([]*ast.File, 0, len(files))

	fset := token.NewFileSet()

	for f := range files {
		parsed, err := parser.ParseFile(
			fset,
			f,
			nil,
			parser.ParseComments|parser.PackageClauseOnly,
		)
		if err != nil {
			return nil, err
		}

		s = append(s, parsed)
	}

	return s, nil
}
