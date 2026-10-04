// Package packageslite implement some functionality
// from the [golang.org/x/tools/go/packages].
//
// [golang.org/x/tools/go/packages]: https://pkg.go.dev/golang.org/x/tools/go/packages
package packageslite

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type Package struct {
	Path        string
	Name        string
	Types       *types.Package
	Syntax      []*ast.File
	Dir         string
	TestImports []string
	TestGoFiles []string
	Info        types.Info

	depOnly bool
}

func (p *Package) Init(fset *token.FileSet, conf *types.Config) error {
	info := types.Info{
		Uses:  make(map[*ast.Ident]types.Object),
		Types: make(map[ast.Expr]types.TypeAndValue),
	}

	checked, err := conf.Check(p.Path, fset, p.Syntax, &info)
	if err != nil {
		return err
	}

	p.Types = checked
	p.Info = info

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

	goPkgMap := make(map[string]*goPackage, len(listed))
	for i := range listed {
		goPkgMap[listed[i].ImportPath] = &listed[i]
	}

	state := &loadState{
		pkgs:    make(map[string]*Package),
		archive: archiveImporter(config.FSet, goPkgMap),
	}

	conf := types.Config{
		// IgnoreFuncBodies:         true,
		DisableUnusedImportCheck: true,
		FakeImportC:              true,
		Importer:                 state,
	}

	for i := range listed {
		if err := state.load(&listed[i], config.FSet, &conf); err != nil {
			return nil, err
		}
	}

	return directPackages(state.pkgs), nil
}

// loadState is the shared state of a single Load call:
// it resolves imports from already type-checked packages
// falling back to compiled export data.
type loadState struct {
	pkgs      map[string]*Package
	archive   types.Importer
	importMap map[string]string
}

var _ types.Importer = (*loadState)(nil)

func (s *loadState) Import(path string) (*types.Package, error) {
	if path == "unsafe" {
		return types.Unsafe, nil
	}

	// Taken from https://github.com/tinygo-org/tinygo/pull/1588/files
	if to, ok := s.importMap[path]; ok && !strings.HasSuffix(to, ".test]") {
		path = to
	}

	if pkg, ok := s.pkgs[path]; ok {
		return pkg.Types, nil
	}

	return s.archive.Import(path)
}

func (s *loadState) load(l *goPackage, fset *token.FileSet, conf *types.Config) error {
	s.importMap = l.ImportMap

	if l.DepOnly {
		pkg, err := loadFromExport(l, s.archive)
		if err != nil {
			return err
		}

		s.pkgs[pkg.Path] = pkg

		return nil
	}

	if l.Error != nil {
		// e.g. build constraints exclude all Go files: nothing to analyze
		if len(l.GoFiles) == 0 {
			return nil
		}

		return fmt.Errorf("%s: %s", l.ImportPath, l.Error.Err)
	}

	pkg, err := l.Package(fset)
	if err != nil {
		return err
	}

	if err := pkg.Init(fset, conf); err != nil {
		return err
	}

	s.pkgs[pkg.Path] = &pkg

	return nil
}

// directPackages returns the non-dependency packages,
// dropping base packages superseded by their "pkg [pkg.test]" variant,
// which go list -test emits as a strict superset of the base package.
func directPackages(pkgs map[string]*Package) []*Package {
	variants := make(map[string]bool)

	for path := range pkgs {
		// only "foo [foo.test]" supersedes "foo"; an external test
		// package "foo_test [foo.test]" supersedes nothing
		if base, ok := testVariantBase(path); ok && path == base+" ["+base+".test]" {
			variants[base] = true
		}
	}

	var direct []*Package

	for _, pkg := range pkgs {
		if pkg.depOnly || variants[pkg.Path] {
			continue
		}

		// normalize the variant path after type checking, purely cosmetic
		if base, ok := testVariantBase(pkg.Path); ok {
			pkg.Path = base
		}

		direct = append(direct, pkg)
	}

	return direct
}

// testVariantBase extracts "pkg" from a go list -test
// variant import path like "pkg [pkg.test]".
func testVariantBase(path string) (string, bool) {
	base, _, ok := strings.Cut(path, " [")

	return base, ok
}

func archiveImporter(fset *token.FileSet, goPkgs map[string]*goPackage) types.Importer {
	return importer.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) {
		if path == "unsafe" {
			return nil, errors.New("unsafe is built-in")
		}

		gp, ok := goPkgs[path]
		if !ok {
			return nil, fmt.Errorf("archive importer: unknown package %q", path)
		}

		if gp.Export == "" {
			return nil, fmt.Errorf("archive importer: no export data for %q", path)
		}

		return os.Open(gp.Export)
	})
}

func loadFromExport(gp *goPackage, imp types.Importer) (*Package, error) {
	typesPkg, err := imp.Import(gp.ImportPath)
	if err != nil {
		return nil, fmt.Errorf("loading export of %s: %w", gp.ImportPath, err)
	}

	return &Package{
		Path:    gp.ImportPath,
		Name:    gp.Name,
		Types:   typesPkg,
		depOnly: true,
	}, nil
}

//nolint:tagliatelle // decodes go list -json output, which is PascalCase
type goPackage struct {
	Dir         string            `json:"Dir"`
	ImportPath  string            `json:"ImportPath"`
	Name        string            `json:"Name"`
	DepOnly     bool              `json:"DepOnly"`
	GoFiles     []string          `json:"GoFiles"`
	CGoFiles    []string          `json:"CGoFiles"`
	Export      string            `json:"Export"`
	Imports     []string          `json:"Imports"`
	ImportMap   map[string]string `json:"ImportMap"`
	TestImports []string          `json:"TestImports"`
	TestGoFiles []string          `json:"TestGoFiles"`
	Incomplete  bool              `json:"Incomplete"`
	Error       *struct {
		Err string `json:"Err"`
	} `json:"Error"`

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
			parser.ParseComments|parser.SkipObjectResolution,
		)
		if err != nil {
			return Package{}, err
		}

		files = append(files, file)
	}

	return Package{
		Path:        gp.ImportPath,
		Name:        gp.Name,
		Types:       types.NewPackage(gp.ImportPath, gp.Name),
		Syntax:      files,
		TestImports: gp.TestImports,
		TestGoFiles: gp.TestGoFiles,
		Dir:         gp.Dir,
		depOnly:     gp.DepOnly,
	}, nil
}

func goList(tags string, patterns ...string) ([]goPackage, error) {
	args := slices.Concat([]string{
		"list",
		"-e",
		"-deps",
		"-test",
		"-export",
		"-buildvcs=false",
		"-pgo=off",
		"-tags",
		tags,
		// TODO: we should only fetch used fields, not everything.
		// NOTE: does not work with with that, for some reason.
		// "-json=Dir,ImportPath,Name,DepOnly,GoFiles,CGoFiles,Imports,ImportMap,Incomplete,Error",
		"-json",
		"--",
	}, patterns)

	//nolint:gosec // variable only affects patterns, safe to use
	cmd := exec.CommandContext(context.Background(), "go", args...)

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
