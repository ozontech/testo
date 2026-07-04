package loader

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"go/token"
	"go/types"
	"io"
	"slices"
	"strings"

	"github.com/ozontech/testo/cmd/testo/internal/packageslite"
	"github.com/ozontech/testo/cmd/testo/internal/typeutil"
	"github.com/ozontech/testo/internal/parse"
)

type LoadError struct {
	FSet        *token.FileSet
	Diagnostics []Diagnostic
}

func (l *LoadError) Error() string {
	msgs := make([]string, 0, len(l.Diagnostics))

	for _, d := range l.Diagnostics {
		var buf bytes.Buffer

		d.Print(&buf, l.FSet)

		msgs = append(msgs, buf.String())
	}

	return strings.Join(msgs, "\n")
}

type Config struct {
	Tags   string
	Testo  string
	Strict bool
}

func Load(cfg Config, patterns ...string) ([]Suite, error) {
	fset := token.NewFileSet()

	pkgs, err := packageslite.Load(packageslite.Config{
		FSet: fset,
		Tags: cfg.Tags,
	}, patterns...)
	if err != nil {
		return nil, err
	}

	var (
		suites      []Suite
		diagnostics []Diagnostic
	)

	for _, pkg := range pkgs {
		if !pkg.Types.Complete() {
			return nil, fmt.Errorf("package %q is not complete", pkg.Name)
		}

		scope := pkg.Types.Scope()

		for _, name := range scope.Names() {
			obj := scope.Lookup(name)

			suite, d, ok := cfg.asSuite(fset, pkg, obj)
			if !ok {
				continue
			}

			diagnostics = append(diagnostics, d...)

			suites = append(suites, suite)
		}
	}

	slices.SortFunc(diagnostics, func(a, b Diagnostic) int {
		return strings.Compare(
			fset.File(a.Pos).Name(),
			fset.File(b.Pos).Name(),
		)
	})

	slices.SortFunc(suites, func(a, b Suite) int {
		return cmp.Compare(a.Name, b.Name)
	})

	if len(diagnostics) > 0 {
		return suites, &LoadError{
			FSet:        fset,
			Diagnostics: diagnostics,
		}
	}

	return suites, nil
}

func (c Config) asSuite(fset *token.FileSet, pkg *packageslite.Package, obj types.Object) (Suite, []Diagnostic, bool) {
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return Suite{}, nil, false
	}

	s, ok := named.Underlying().(*types.Struct)
	if !ok {
		return Suite{}, nil, false
	}

	var (
		t    T
		hasT bool
	)

	for f := range s.Fields() {
		t, hasT = c.asT(f)
		if hasT {
			break
		}
	}

	if !hasT {
		return Suite{}, nil, false
	}

	suite := Suite{
		FSet:    fset,
		Pos:     obj.Pos(),
		Package: pkg,
		Name:    named.Obj().Name(),
		T:       t,
	}

	cases, diagnostics, fatal := c.collectCases(named)
	if fatal {
		return suite, diagnostics, true
	}

	for m := range named.Methods() {
		name := m.Name()

		const prefix = "Test"

		if !strings.HasPrefix(name, prefix) {
			continue
		}

		if !parse.IsTest(name, prefix) {
			diagnostics = append(diagnostics, Diagnostic{
				Pos: m.Pos(),
				Issue: MalformedName(
					fmt.Sprintf("first letter after %q in %q must not be lowercase", prefix, name),
				),
			})

			continue
		}

		sig := m.Signature()

		if sig.Results().Len() > 0 {
			diagnostics = append(diagnostics, Diagnostic{
				Pos:   m.Pos(),
				Issue: InvalidSignature(name + " must not return values"),
			})

			continue
		}

		params := sig.Params()

		switch params.Len() {
		case 1:
			in := params.At(0)

			if !types.Identical(in.Type(), suite.T.Type) {
				diagnostics = append(diagnostics, Diagnostic{
					Pos: m.Pos(),
					Issue: InvalidSignature(
						fmt.Sprintf(
							"%s must accept %s, got %s",
							name,
							typeutil.Format(suite.T.Type),
							typeutil.Format(in.Type()),
						),
					),
				})
			}

			suite.Tests = append(suite.Tests, SuiteTest{
				Name: name,
			})

		case 2:
			first := params.At(0)

			if !types.Identical(first.Type(), suite.T.Type) {
				diagnostics = append(diagnostics, Diagnostic{
					Pos: m.Pos(),
					Issue: InvalidSignature(
						fmt.Sprintf(
							"%s must accept %s, got %s",
							name,
							typeutil.Format(suite.T.Type),
							typeutil.Format(first.Type()),
						),
					),
				})

				continue
			}

			second := params.At(1)

			params, ok := second.Type().Underlying().(*types.Struct)
			if !ok {
				diagnostics = append(diagnostics, Diagnostic{
					Pos: m.Pos(),
					Issue: InvalidSignature(
						fmt.Sprintf(
							"%s must accept struct as second parameter, got %s",
							name,
							second.Type(),
						),
					),
				})

				continue
			}

			var invalidParams bool

			var parameters []Parameter

			for f := range params.Fields() {
				if !f.Exported() {
					diagnostics = append(diagnostics, Diagnostic{
						Pos: m.Pos(),
						Issue: InvalidSignature(
							fmt.Sprintf("%s parameters must be exported, got %s", name, f.Name()),
						),
					})

					invalidParams = true

					continue
				}

				forParam, ok := cases[f.Name()]
				if !ok {
					diagnostics = append(diagnostics, Diagnostic{
						Pos: m.Pos(),
						Issue: InvalidSignature(
							fmt.Sprintf("%s requires unknown parameter %s", name, f.Name()),
						),
					})

					invalidParams = true

					continue
				}

				if !types.AssignableTo(forParam.Type, f.Type()) {
					diagnostics = append(diagnostics, Diagnostic{
						Pos: m.Pos(),
						Issue: InvalidSignature(
							fmt.Sprintf(
								"%s requires param %s to be of type %s, have %s",
								name,
								f.Name(),
								f.Type(),
								forParam.Type,
							),
						),
					})

					invalidParams = true

					continue
				}

				parameters = append(parameters, Parameter{
					Name: f.Name(),
					Type: f.Type(),
				})
			}

			suite.Tests = append(suite.Tests, SuiteTest{
				Name:         name,
				Parametrized: true,
				Parameters:   parameters,
			})

			if invalidParams {
				continue
			}

		default:
			diagnostics = append(diagnostics, Diagnostic{
				Pos: m.Pos(),
				Issue: InvalidSignature(
					fmt.Sprintf(
						"%s must accept either 1 or 2 parameters, got %d",
						name,
						params.Len(),
					),
				),
			})
		}
	}

	if c.Strict && len(diagnostics) == 0 && len(suite.Tests) == 0 {
		diagnostics = append(diagnostics, Diagnostic{
			Pos:   obj.Pos(),
			Issue: TestsMissing(fmt.Sprintf("suite %s has no tests", suite.Name)),
		})
	}

	return suite, diagnostics, true
}

func (c Config) asT(f *types.Var) (T, bool) {
	if !f.Embedded() {
		return T{}, false
	}

	if f.Name() != "Suite" {
		return T{}, false
	}

	suite, ok := f.Type().(*types.Named)
	if !ok {
		return T{}, false
	}

	if suite.Obj().Pkg().Path() != c.Testo {
		return T{}, false
	}

	aStruct := suite.Underlying().(*types.Struct)
	field := aStruct.Field(0)
	array := field.Type().Underlying().(*types.Array)
	pointer := array.Elem().Underlying().(*types.Pointer)

	elem := pointer.Elem()

	return T{Type: elem}, true
}

type Cases map[string]Case

type Case struct {
	Name string
	Type types.Type
}

func (c Config) collectCases(
	suite *types.Named,
) (cases Cases, diagnostics []Diagnostic, fatal bool) {
	cases = make(Cases)

	const prefix = "Cases"

	for m := range suite.Methods() {
		if !strings.HasPrefix(m.Name(), prefix) {
			continue
		}

		if !parse.IsTest(m.Name(), prefix) {
			diagnostics = append(diagnostics, Diagnostic{
				Pos: m.Pos(),
				Issue: MalformedName(
					fmt.Sprintf(
						"first letter after %q in %q must not be lowercase",
						prefix,
						m.Name(),
					),
				),
			})

			fatal = true

			continue
		}

		sig := m.Signature()

		if sig.Params().Len() != 0 {
			diagnostics = append(diagnostics, Diagnostic{
				Pos:   m.Pos(),
				Issue: InvalidSignature(m.Name() + " must not accept parameters"),
			})

			fatal = true

			continue
		}

		results := sig.Results()

		if results.Len() != 1 {
			diagnostics = append(diagnostics, Diagnostic{
				Pos:   m.Pos(),
				Issue: InvalidSignature(m.Name() + " must return exactly one result"),
			})

			fatal = true

			continue
		}

		out := results.At(0).Type().Underlying()

		s, ok := out.(*types.Slice)
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{
				Pos: m.Pos(),
				Issue: InvalidSignature(
					fmt.Sprintf("%s must return a slice, got %s", m.Name(), out),
				),
			})

			fatal = true

			continue
		}

		name := strings.TrimPrefix(m.Name(), prefix)
		elem := s.Elem()

		if private, numFields := findPrivateFields(elem); len(private) > 0 {
			if len(private) == numFields {
				diagnostics = append(diagnostics, Diagnostic{
					Pos: private[0].Pos(),
					Issue: PrivateField(fmt.Sprintf(
						"type returned by %s%s contains only private fields",
						prefix, name,
					)),
				})
			} else if c.Strict {
				diagnostics = append(diagnostics, Diagnostic{
					Pos: private[0].Pos(),
					Issue: PrivateField(fmt.Sprintf(
						"type returned by %s%s contains private field %q",
						prefix, name, private[0].Name(),
					)),
				})
			}
		}

		cases[name] = Case{
			Name: name,
			Type: elem,
		}
	}

	return cases, diagnostics, fatal
}

func findPrivateFields(t types.Type) ([]*types.Var, int) {
	s, ok := t.Underlying().(*types.Struct)
	if !ok {
		return nil, 0
	}

	private := make([]*types.Var, 0, s.NumFields())

	for f := range s.Fields() {
		if !f.Exported() {
			private = append(private, f)
		}
	}

	return private, s.NumFields()
}

type Diagnostic struct {
	Pos   token.Pos
	Issue Issue
}

func (d Diagnostic) JSON(w io.Writer, set *token.FileSet) {
	type Entry struct {
		File    string `json:"file"`
		Line    int    `json:"line"`
		Kind    string `json:"kind"`
		Message string `json:"message"`
	}

	file := set.File(d.Pos)

	entry := Entry{
		File:    file.Name(),
		Line:    file.Line(d.Pos),
		Kind:    d.Issue.Kind(),
		Message: d.Issue.Message(),
	}

	_ = json.NewEncoder(w).Encode(entry)
}

func (d Diagnostic) Print(w io.Writer, set *token.FileSet) {
	file := set.File(d.Pos)
	line := file.Line(d.Pos)

	fmt.Fprintf(w, "%s:%d: %s\n", file.Name(), line, d.Issue.String())
}

type Issue interface {
	fmt.Stringer

	Kind() string
	Message() string

	issue()
}

type PrivateField string

func (pf PrivateField) Kind() string {
	return "private field"
}

func (pf PrivateField) Message() string {
	return string(pf)
}

func (pf PrivateField) String() string {
	return pf.Kind() + ": " + pf.Message()
}

func (PrivateField) issue() {}

type MalformedName string

func (mn MalformedName) Kind() string {
	return "malformed name"
}

func (mn MalformedName) Message() string {
	return string(mn)
}

func (mn MalformedName) String() string {
	return mn.Kind() + ": " + mn.Message()
}

func (MalformedName) issue() {}

type InvalidSignature string

func (is InvalidSignature) Kind() string {
	return "invalid signature"
}

func (is InvalidSignature) Message() string {
	return string(is)
}

func (is InvalidSignature) String() string {
	return is.Kind() + ": " + is.Message()
}

func (InvalidSignature) issue() {}

type TestsMissing string

func (tm TestsMissing) Kind() string {
	return "tests missing"
}

func (tm TestsMissing) Message() string {
	return string(tm)
}

func (tm TestsMissing) String() string {
	return tm.Kind() + ": " + tm.Message()
}

func (TestsMissing) issue() {}

type Suite struct {
	FSet    *token.FileSet
	Pos     token.Pos
	Package *packageslite.Package
	Name    string
	Tests   []SuiteTest
	T       T
}

type SuiteRunner struct {
	Dir  string
	Name string
}

type T struct {
	Type types.Type
}

type SuiteTest struct {
	Name         string
	Parametrized bool
	Parameters   []Parameter
}

type Parameter struct {
	Name string
	Type types.Type
}

func unquote(s string) string {
	return strings.Trim(s, `"'`)
}
