package loader

import (
	"bytes"
	"cmp"
	"context"
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

	runners map[string]map[SuiteRunner]struct{}
}

// deriveTags collects build tags from module sources,
// excluding tags satisfied by the toolchain itself.
func deriveTags() (string, error) {
	add, remove, err := BuildTags(false)
	if err != nil {
		return "", fmt.Errorf("derive build tags: %w", err)
	}

	tags := make([]string, 0, len(add))

	for k := range add {
		if _, ok := remove[k]; ok {
			continue
		}

		// tags satisfied by the toolchain itself (GOOS, GOARCH, go1.x, ...)
		// must not be passed to -tags: enabling e.g. both linux and windows
		// would pull mutually exclusive files into type checking
		if isImplicitTag(k) {
			continue
		}

		tags = append(tags, k)
	}

	slices.Sort(tags)

	return strings.Join(tags, ","), nil
}

func Load(cfg Config, patterns ...string) ([]Suite, error) {
	if cfg.Tags == "" {
		tags, err := deriveTags()
		if err != nil {
			return nil, err
		}

		cfg.Tags = tags
	}

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

			suite, d, ok := cfg.asSuite(fset, pkg, pkgs, obj)
			if !ok {
				continue
			}

			diagnostics = append(diagnostics, d...)

			suites = append(suites, suite)
		}
	}

	slices.SortFunc(diagnostics, func(a, b Diagnostic) int {
		return cmp.Or(
			strings.Compare(
				fset.File(a.Pos).Name(),
				fset.File(b.Pos).Name(),
			),
			cmp.Compare(a.Pos, b.Pos),
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

func (c *Config) asSuite(
	fset *token.FileSet,
	pkg *packageslite.Package,
	pkgs []*packageslite.Package,
	obj types.Object,
) (Suite, []Diagnostic, bool) {
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
		Type:    obj.Type(),
	}

	suite.Runners = func(ctx context.Context) ([]SuiteRunner, error) {
		return c.loadRunners(ctx, fset, suite, pkgs)
	}

	cases, diagnostics, fatal := c.collectCases(named)
	if fatal {
		return suite, diagnostics, true
	}

	for m := range named.Methods() {
		test, d, ok := c.asSuiteTest(m, suite.T.Type, cases)

		diagnostics = append(diagnostics, d...)

		if ok {
			suite.Tests = append(suite.Tests, test)
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

// asSuiteTest validates a TestXxx suite method and describes it as a SuiteTest.
func (c *Config) asSuiteTest(
	m *types.Func,
	tType types.Type,
	cases Cases,
) (SuiteTest, []Diagnostic, bool) {
	name := m.Name()

	const prefix = "Test"

	if !strings.HasPrefix(name, prefix) {
		return SuiteTest{}, nil, false
	}

	if !parse.IsTest(name, prefix) {
		return SuiteTest{}, []Diagnostic{{
			Pos: m.Pos(),
			Issue: MalformedName(
				fmt.Sprintf("first letter after %q in %q must not be lowercase", prefix, name),
			),
		}}, false
	}

	sig := m.Signature()

	if sig.Results().Len() > 0 {
		return SuiteTest{}, []Diagnostic{{
			Pos:   m.Pos(),
			Issue: InvalidSignature(name + " must not return values"),
		}}, false
	}

	params := sig.Params()

	if params.Len() < 1 || params.Len() > 2 {
		return SuiteTest{}, []Diagnostic{{
			Pos: m.Pos(),
			Issue: InvalidSignature(
				fmt.Sprintf(
					"%s must accept either 1 or 2 parameters, got %d",
					name,
					params.Len(),
				),
			),
		}}, false
	}

	if in := params.At(0); !types.Identical(in.Type(), tType) {
		return SuiteTest{}, []Diagnostic{{
			Pos: m.Pos(),
			Issue: InvalidSignature(
				fmt.Sprintf(
					"%s must accept %s, got %s",
					name,
					typeutil.Format(tType),
					typeutil.Format(in.Type()),
				),
			),
		}}, false
	}

	if params.Len() == 1 {
		return SuiteTest{
			Name: name,
			Pos:  m.Pos(),
		}, nil, true
	}

	return asParametrizedTest(m, cases)
}

// asParametrizedTest validates the second (parameters struct)
// argument of a TestXxx suite method against collected cases.
func asParametrizedTest(m *types.Func, cases Cases) (SuiteTest, []Diagnostic, bool) {
	name := m.Name()
	second := m.Signature().Params().At(1)

	params, ok := second.Type().Underlying().(*types.Struct)
	if !ok {
		return SuiteTest{}, []Diagnostic{{
			Pos: m.Pos(),
			Issue: InvalidSignature(
				fmt.Sprintf(
					"%s must accept struct as second parameter, got %s",
					name,
					second.Type(),
				),
			),
		}}, false
	}

	var diagnostics []Diagnostic

	var parameters []Parameter

	for f := range params.Fields() {
		if !f.Exported() {
			diagnostics = append(diagnostics, Diagnostic{
				Pos: m.Pos(),
				Issue: InvalidSignature(
					fmt.Sprintf("%s parameters must be exported, got %s", name, f.Name()),
				),
			})

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

			continue
		}

		parameters = append(parameters, Parameter{
			Name: f.Name(),
			Type: f.Type(),
		})
	}

	if len(diagnostics) > 0 {
		return SuiteTest{}, diagnostics, false
	}

	return SuiteTest{
		Pos:          m.Pos(),
		Name:         name,
		Parametrized: true,
		Parameters:   parameters,
	}, nil, true
}

func (c *Config) asT(f *types.Var) (T, bool) {
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

	if pkg := suite.Obj().Pkg(); pkg == nil || pkg.Path() != c.Testo {
		return T{}, false
	}

	// testo.Suite[T] is struct{ _ [0]*T }; extract T without
	// panicking if the layout differs (e.g. CLI/testo version skew)
	aStruct, ok := suite.Underlying().(*types.Struct)
	if !ok || aStruct.NumFields() == 0 {
		return T{}, false
	}

	array, ok := aStruct.Field(0).Type().Underlying().(*types.Array)
	if !ok {
		return T{}, false
	}

	pointer, ok := array.Elem().Underlying().(*types.Pointer)
	if !ok {
		return T{}, false
	}

	return T{Type: pointer.Elem()}, true
}

type Cases map[string]Case

type Case struct {
	Name string
	Type types.Type
}

func (c *Config) collectCases(
	suite *types.Named,
) (cases Cases, diagnostics []Diagnostic, fatal bool) {
	cases = make(Cases)

	const prefix = "Cases"

	for m := range suite.Methods() {
		if !strings.HasPrefix(m.Name(), prefix) {
			continue
		}

		aCase, d, ok := c.asCase(m)

		diagnostics = append(diagnostics, d...)

		if !ok {
			fatal = true

			continue
		}

		cases[aCase.Name] = aCase
	}

	return cases, diagnostics, fatal
}

// asCase validates a CasesXxx suite method and describes it as a Case.
// Non-ok diagnostics are fatal; an ok Case may still carry warnings.
func (c *Config) asCase(m *types.Func) (Case, []Diagnostic, bool) {
	const prefix = "Cases"

	if !parse.IsTest(m.Name(), prefix) {
		return Case{}, []Diagnostic{{
			Pos: m.Pos(),
			Issue: MalformedName(
				fmt.Sprintf(
					"first letter after %q in %q must not be lowercase",
					prefix,
					m.Name(),
				),
			),
		}}, false
	}

	sig := m.Signature()

	if sig.Params().Len() != 0 {
		return Case{}, []Diagnostic{{
			Pos:   m.Pos(),
			Issue: InvalidSignature(m.Name() + " must not accept parameters"),
		}}, false
	}

	results := sig.Results()

	if results.Len() != 1 {
		return Case{}, []Diagnostic{{
			Pos:   m.Pos(),
			Issue: InvalidSignature(m.Name() + " must return exactly one result"),
		}}, false
	}

	out := results.At(0).Type().Underlying()

	s, ok := out.(*types.Slice)
	if !ok {
		return Case{}, []Diagnostic{{
			Pos: m.Pos(),
			Issue: InvalidSignature(
				fmt.Sprintf("%s must return a slice, got %s", m.Name(), out),
			),
		}}, false
	}

	name := strings.TrimPrefix(m.Name(), prefix)
	elemType := s.Elem()

	var diagnostics []Diagnostic

	if private, numFields := findPrivateFields(elemType); len(private) > 0 {
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

	return Case{
		Name: name,
		Type: elemType,
	}, diagnostics, true
}

func findPrivateFields(t types.Type) (private []*types.Var, numFields int) {
	s, ok := t.Underlying().(*types.Struct)
	if !ok {
		return nil, 0
	}

	private = make([]*types.Var, 0, s.NumFields())

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

func (d Diagnostic) JSON(w io.Writer, set *token.FileSet) error {
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

	return json.NewEncoder(w).Encode(entry)
}

func (d Diagnostic) Print(w io.Writer, set *token.FileSet) {
	file := set.File(d.Pos)
	line := file.Line(d.Pos)

	fmt.Fprintf(w, "%s:%d: %s", file.Name(), line, d.Issue.String())
}

func (d Diagnostic) Println(w io.Writer, set *token.FileSet) {
	d.Print(w, set)
	fmt.Fprintln(w)
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
	Runners func(ctx context.Context) ([]SuiteRunner, error)
	Type    types.Type
}

func (s Suite) ID() string {
	return s.Package.Path + "." + s.Name
}

type SuiteRunner struct {
	Dir  string
	Name string
	Tags string
}

type T struct {
	Type types.Type
}

type SuiteTest struct {
	Name         string
	Parametrized bool
	Parameters   []Parameter
	Pos          token.Pos
}

type Parameter struct {
	Name string
	Type types.Type
}
