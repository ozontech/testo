// Package packageslite implement some functionality
// from the [golang.org/x/tools/go/packages].
//
// [golang.org/x/tools/go/packages]: https://pkg.go.dev/golang.org/x/tools/go/packages
package packageslite

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type Package struct {
	Path   string
	Name   string
	Types  *types.Package
	Syntax []*ast.File

	depOnly bool
}

func (p *Package) Init(fset *token.FileSet, conf *types.Config) error {
	checked, err := conf.Check(p.Path, fset, p.Syntax, nil)
	if err != nil {
		return err
	}

	p.Types = checked

	return nil
}

type Config struct {
	FSet *token.FileSet
	Tags string
}

func Load(config Config, patterns ...string) ([]*Package, error) {
	listed, err := goList(config.Tags, patterns...)
	if err != nil {
		return nil, err
	}

	pkgs := make(map[string]*Package)

	var importMap map[string]string

	conf := types.Config{
		IgnoreFuncBodies:         true,
		DisableUnusedImportCheck: true,
		FakeImportC:              true,
		Importer: importerFunc(func(path string) (*types.Package, error) {
			if path == "unsafe" {
				return types.Unsafe, nil
			}

			if importMap != nil {
				// Taken from https://github.com/tinygo-org/tinygo/pull/1588/files
				if to, ok := importMap[path]; ok && !strings.HasSuffix(to, ".test]") {
					path = to
				}
			}

			pkg, ok := pkgs[path]
			if !ok {
				return nil, errors.New("package not found")
			}

			return pkg.Types, nil
		}),
	}

	for _, l := range listed {
		importMap = l.ImportMap

		pkg, err := l.Package(config.FSet)
		if err != nil {
			return nil, err
		}

		err = pkg.Init(config.FSet, &conf)
		if err != nil {
			return nil, err
		}

		pkgs[pkg.Path] = &pkg
	}

	var direct []*Package

	for _, pkg := range pkgs {
		if pkg.depOnly {
			continue
		}

		direct = append(direct, pkg)
	}

	return direct, nil
}

var _ types.Importer = (*importerFunc)(nil)

type importerFunc func(path string) (*types.Package, error)

func (i importerFunc) Import(path string) (*types.Package, error) {
	return i(path)
}

type goPackage struct {
	Dir        string
	ImportPath string
	Name       string
	DepOnly    bool
	GoFiles    []string
	CGoFiles   []string
	Imports    []string
	ImportMap  map[string]string
	Incomplete bool
	Error      *struct {
		Err string
	}

	order int
}

func (gp goPackage) Package(fset *token.FileSet) (Package, error) {
	names := slices.Concat(
		gp.GoFiles,
		gp.CGoFiles,
	)

	files := make([]*ast.File, 0, len(names))

	for _, n := range names {
		file, err := parser.ParseFile(
			fset,
			filepath.Join(gp.Dir, n),
			nil,
			parser.ParseComments,
		)
		if err != nil {
			return Package{}, err
		}

		files = append(files, file)
	}

	return Package{
		Path:    gp.ImportPath,
		Name:    gp.Name,
		Types:   types.NewPackage(gp.ImportPath, gp.Name),
		Syntax:  files,
		depOnly: gp.DepOnly,
	}, nil
}

func goList(tags string, patterns ...string) ([]goPackage, error) {
	args := []string{
		"list",
		"-e",
		"-deps",
		"-test",
		"-export",
		"-buildvcs=false",
		"-pgo=off",
		"-tags",
		tags,
		// "-json=Dir,ImportPath,Name,DepOnly,GoFiles,CGoFiles,Imports,ImportMap,Incomplete,Error",
		"-json",
		"--",
	}

	args = append(args, patterns...)

	//nolint:gosec // variable only affects patterns, safe to use
	cmd := exec.Command("go", args...)

	cmd.Env = os.Environ()

	out, err := cmd.Output()
	if err != nil {
		var errExit *exec.ExitError

		if errors.As(err, &errExit) {
			return nil, fmt.Errorf("go list: %w", err)
		}

		return nil, err
	}

	var packages []goPackage

	dec := json.NewDecoder(bytes.NewReader(out))

	var i int

	for dec.More() {
		i++

		var pkg goPackage

		err = dec.Decode(&pkg)
		if err != nil {
			return nil, err
		}

		if strings.HasSuffix(pkg.ImportPath, ".test") {
			continue
		}

		if len(pkg.Imports) > 0 {
			pkg.order = i
		}

		packages = append(packages, pkg)
	}

	slices.SortStableFunc(packages, func(a, b goPackage) int {
		return cmp.Compare(a.order, b.order)
	})

	return packages, nil
}
