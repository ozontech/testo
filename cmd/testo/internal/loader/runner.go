package loader

import (
	"context"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"github.com/ozontech/testo/cmd/testo/internal/packageslite"
	"github.com/ozontech/testo/internal/parse"
)

func (c *Config) loadRunners(
	ctx context.Context,
	fset *token.FileSet,
	suite Suite,
	pkgs []*packageslite.Package,
) ([]SuiteRunner, error) {
	var runners []SuiteRunner

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			tokenFile := fset.File(file.Pos())

			if !strings.HasSuffix(tokenFile.Name(), "_test.go") {
				continue
			}

			var testName string

			ast.Inspect(file, func(n ast.Node) bool {
				if f, ok := n.(*ast.FuncDecl); ok {
					if parse.IsTest(f.Name.Name, "Test") {
						testName = f.Name.Name
					}

					return true
				}

				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				fun := pkg.Info.Uses[funcIdent(call)]
				if fun == nil {
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if ok {
						fun = pkg.Info.Uses[sel.Sel]
					}
				}

				if fun == nil {
					return true
				}

				fn, ok := fun.(*types.Func)
				if !ok {
					return true
				}

				if fn.Pkg() == nil || fn.Pkg().Path() != c.Testo || fn.Name() != "RunSuite" {
					return true
				}

				sig := pkg.Info.Types[call.Fun].Type.(*types.Signature)

				params := sig.Params()

				if params.Len() == 0 {
					return true
				}

				identical := types.Identical(elem(params.At(1).Type()), elem(suite.Type))

				if identical {
					runners = append(runners, SuiteRunner{
						Name: testName,
						Dir:  filepath.Dir(tokenFile.Name()),
					})
				}

				return true
			})
		}
	}

	return runners, nil
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

func elem(a types.Type) types.Type {
	if ptr, ok := a.(*types.Pointer); ok {
		return elem(ptr.Elem())
	}

	return a
}
