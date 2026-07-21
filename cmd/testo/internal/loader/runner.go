package loader

import (
	"context"
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ozontech/testo/cmd/testo/internal/packageslite"
	"github.com/ozontech/testo/internal/parse"
)

func (c *Config) loadRunners(
	_ context.Context,
	fset *token.FileSet,
	suite Suite,
	pkgs []*packageslite.Package,
) ([]SuiteRunner, error) {
	if c.runners == nil {
		c.scanRunners(fset, pkgs)
	}

	runners := slices.Collect(maps.Keys(c.runners[suiteKey(suite.Type)]))

	slices.SortFunc(runners, func(a, b SuiteRunner) int {
		return strings.Compare(a.Dir+"\x00"+a.Name, b.Dir+"\x00"+b.Name)
	})

	return runners, nil
}

// scanRunners finds all testo.RunSuite calls in test files
// and records them keyed by the suite type they run.
func (c *Config) scanRunners(fset *token.FileSet, pkgs []*packageslite.Package) {
	c.runners = make(map[string]map[SuiteRunner]struct{})

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			tokenFile := fset.File(file.Pos())

			if !strings.HasSuffix(tokenFile.Name(), "_test.go") {
				continue
			}

			var testName string

			ast.Inspect(file, func(n ast.Node) bool {
				if f, ok := n.(*ast.FuncDecl); ok {
					// only top-level test functions can be addressed with go test -run;
					// methods (e.g. suite tests) and helpers must not be attributed
					if f.Recv == nil && parse.IsTest(f.Name.Name, "Test") {
						testName = f.Name.Name
					} else {
						testName = ""
					}

					return true
				}

				suiteType, ok := c.runSuiteArg(pkg, n)
				if !ok {
					return true
				}

				if testName == "" {
					// RunSuite is called outside of a test function (e.g. in a helper);
					// there is no test name to attribute the call to
					return true
				}

				c.addRunner(suiteKey(suiteType), SuiteRunner{
					Name: testName,
					Dir:  filepath.Dir(tokenFile.Name()),
					Tags: c.Tags,
				})

				return true
			})
		}
	}
}

// runSuiteArg states whether the node is a testo.RunSuite
// call and returns the suite type it is called with.
func (c *Config) runSuiteArg(pkg *packageslite.Package, n ast.Node) (types.Type, bool) {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return nil, false
	}

	fun := pkg.Info.Uses[funcIdent(call)]
	if fun == nil {
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if ok {
			fun = pkg.Info.Uses[sel.Sel]
		}
	}

	if fun == nil {
		return nil, false
	}

	fn, ok := fun.(*types.Func)
	if !ok {
		return nil, false
	}

	if fn.Pkg() == nil || fn.Pkg().Path() != c.Testo || fn.Name() != "RunSuite" {
		return nil, false
	}

	sig, ok := pkg.Info.Types[call.Fun].Type.(*types.Signature)
	if !ok {
		return nil, false
	}

	params := sig.Params()

	if params.Len() < 2 {
		return nil, false
	}

	return params.At(1).Type(), true
}

func (c *Config) addRunner(key string, runner SuiteRunner) {
	if _, ok := c.runners[key]; !ok {
		c.runners[key] = make(map[SuiteRunner]struct{})
	}

	c.runners[key][runner] = struct{}{}
}

func funcIdent(call *ast.CallExpr) *ast.Ident {
	switch f := call.Fun.(type) {
	case *ast.Ident:
		return f

	case *ast.SelectorExpr:
		return f.Sel

	default:
		return nil
	}
}

// suiteKey identifies a suite type by normalized package path and type name,
// so that a suite and its RunSuite callers agree even when they were
// type-checked in different go list -test package variants.
func suiteKey(t types.Type) string {
	t = elem(t)

	named, ok := t.(*types.Named)
	if !ok {
		return t.String()
	}

	obj := named.Obj()

	var path string

	if pkg := obj.Pkg(); pkg != nil {
		path, _, _ = strings.Cut(pkg.Path(), " [")
	}

	return path + "." + obj.Name()
}

func elem(a types.Type) types.Type {
	if ptr, ok := a.(*types.Pointer); ok {
		return elem(ptr.Elem())
	}

	return a
}
