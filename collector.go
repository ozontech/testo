package testo

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/ozontech/testo/internal/pragma"
	"github.com/ozontech/testo/internal/testnamer"
	"github.com/ozontech/testo/testoplugin"
	"github.com/ozontech/testo/testoreflect"
)

type (
	suiteTest[Suite suite[T], T CommonT] struct {
		Name string
		Info testoreflect.TestInfo
		Run  func(Suite, T)
	}

	suiteCase[Suite suite[T], T CommonT] struct {
		Provides reflect.Type
		Func     func(Suite) []reflect.Value
	}
)

var _ testoplugin.PlannedTest = (*plannedSuiteTest[Suite[*T], *T])(nil)

type plannedSuiteTest[Suite suite[T], T CommonT] struct {
	inner annotatedSuiteTest[Suite, T]
}

func (plannedSuiteTest[Suite, T]) TestoInternal(pragma.DoNotImplement) {}

func (t plannedSuiteTest[Suite, T]) Info() testoreflect.TestInfo {
	return t.inner.Info
}

func (t plannedSuiteTest[Suite, T]) Annotations() []testoplugin.Option {
	return slices.Clone(t.inner.Options)
}

// isTest states whether name is a valid test name (or other type, according to prefix).
//
// It checks if the next character after prefix is uppercase.
//
//	TestFoo    => true
//	Test       => true
//	TestfooBar => false
func isTest(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}

	// "Test" is ok
	if len(name) == len(prefix) {
		return true
	}

	r, _ := utf8.DecodeRuneInString(name[len(prefix):])

	return !unicode.IsLower(r)
}

func suiteCasesOf[Suite suite[T], T CommonT](tb testing.TB) map[string]suiteCase[Suite, T] {
	tb.Helper()

	vt := reflect.TypeFor[Suite]()

	cases := make(map[string]suiteCase[Suite, T])

	for i := range vt.NumMethod() {
		method := vt.Method(i)

		const prefix = "Cases"

		if !isTest(method.Name, prefix) {
			continue
		}

		name := strings.TrimPrefix(method.Name, prefix)

		if name == "" {
			continue
		}

		isValidIn := method.Type.NumIn() == 1
		isValidOut := method.Type.NumOut() == 1 && method.Type.Out(0).Kind() == reflect.Slice

		if !isValidIn || !isValidOut {
			tb.Fatalf(
				"testo: wrong signature for %[1]s.%[2]s, must be: func (%[1]s) %[2]s() []...",
				reflect.TypeFor[Suite](), method.Name, tb,
			)
		}

		cases[name] = suiteCase[Suite, T]{
			Provides: method.Type.Out(0).Elem(),
			Func: func(s Suite) []reflect.Value {
				var suite reflect.Value

				if method.Type.In(0).Kind() == reflect.Pointer &&
					reflect.TypeOf(s).Kind() != reflect.Pointer {
					suite = reflect.ValueOf(&s)
				} else {
					suite = reflect.ValueOf(s)
				}

				slice := method.Func.Call([]reflect.Value{suite})[0]

				values := make([]reflect.Value, 0, slice.Len())

				for i := range slice.Len() {
					v := slice.Index(i)

					values = append(values, v)
				}

				return values
			},
		}
	}

	return cases
}

// suiteTests contains all the suite tests.
//
// While regular tests are ready to be run,
// parametrized tests are tricky.
//
// We can't know how many permutations (hence number of tests)
// they will have until we receive all values for each case by calling CasesXXX funcs.
// However, we can't do that before running the BeforeAll hooks - cases funcs may
// depend on in being run first.
//
// But we should not run any hooks until we are sure that tests are correct
// and no error should be raised.
//
// That's why we statically analyze parametrized tests signatures,
// but delay the actual collection for later.
type suiteTests[Suite suite[T], T CommonT] struct {
	Regular      []suiteTest[Suite, T]
	Parametrized []suiteTestParametrized[Suite, T]
}

type annotatedSuiteTest[Suite suite[T], T CommonT] struct {
	suiteTest[Suite, T]

	// Options to pass specifically for this test.
	Options []testoplugin.Option

	Configure func(*testoT)
}

// Collect all suite tests.
//
// Suite instance is required here to get
// parameter cases (CasesXXX funcs), not to invoke the actual tests.
func (st suiteTests[Suite, T]) Collect(
	tb testing.TB,
	s Suite,
	name func(string) string,
) []annotatedSuiteTest[Suite, T] {
	tb.Helper()

	tests := make([]annotatedSuiteTest[Suite, T], 0, len(st.Regular))

	for _, r := range st.Regular {
		tests = append(tests, annotatedSuiteTest[Suite, T]{
			suiteTest: r,
			Options:   annotationsFor(getID[Suite](reflect.ValueOf(r.Run))),
		})
	}

	for _, p := range st.Parametrized {
		cases := p.Tests(tb, s)

		tests = append(tests, cases...)
	}

	// special case for [Test] and [RunTest].
	//
	// NOTE(metafates): future "special" suites should be handled here in a type switch.
	if s, ok := any(s).(singleton[T]); ok {
		tests = append(tests, annotatedSuiteTest[Suite, T]{
			suiteTest: suiteTest[Suite, T]{
				Name: s.name,
				Info: testoreflect.RegularTestInfo{
					Name:        name(s.name),
					RawBaseName: s.name,
					Level:       1,
					FuncPC:      reflect.ValueOf(s.test).Pointer(),
				},
				Run: func(_ Suite, t T) { s.test(t) },
			},
			Options: s.options,
			Configure: func(tt *testoT) {
				tt.propagateParallel = true
			},
		})
	}

	return tests
}

type testsCollector[Suite suite[T], T CommonT] struct {
	CallerName string
	TestNamer  *testnamer.Namer
}

func (tc *testsCollector[Suite, T]) testName(base string) string {
	return tc.TestNamer.Name(tc.CallerName, base)
}

//nolint:cyclop,funlen,gocognit // splitting it would make it even more complex
func (tc *testsCollector[Suite, T]) Collect(
	tb testing.TB,
) suiteTests[Suite, T] {
	tb.Helper()

	cases := suiteCasesOf[Suite](tb)

	suiteTyp := reflect.TypeFor[Suite]()

	var tests suiteTests[Suite, T]

	for i := range suiteTyp.NumMethod() {
		method := suiteTyp.Method(i)

		if !isTest(method.Name, "Test") {
			continue
		}

		raiseWrongSignatureError := func() {
			tb.Helper()

			//nolint:lll // it's a long message
			tb.Fatalf(
				"testo: wrong signature for (%[1]s).%[2]s, must be: func (%[1]s).%[2]s(%[3]s) or func (%[1]s).%[2]s(%[3]s, struct{...})",
				suiteTyp,
				method.Name,
				reflect.TypeFor[T](),
			)
		}

		if method.Type.NumOut() != 0 {
			raiseWrongSignatureError()
		}

		if method.Type.NumIn() < 2 {
			raiseWrongSignatureError()
		}

		if method.Type.In(1) != reflect.TypeFor[T]() {
			raiseWrongSignatureError()
		}

		switch method.Type.NumIn() {
		default:
			raiseWrongSignatureError()

		case 2: // regular test - (Suite, T)
			if !flagMethod.MatchString(method.Name) {
				continue
			}

			tests.Regular = append(tests.Regular, suiteTest[Suite, T]{
				Name: method.Name,
				Info: testoreflect.RegularTestInfo{
					Name:        tc.testName(method.Name),
					RawBaseName: method.Name,
					Level:       1,
					FuncPC:      method.Func.Pointer(),
				},
				Run: method.Func.Interface().(func(Suite, T)),
			})

		case 3: // parametrized test - (Suite, T, Params)
			param := method.Type.In(2)

			if param.Kind() != reflect.Struct {
				raiseWrongSignatureError()
			}

			requiredCases := make(map[string]suiteCase[Suite, T])

			for i := range param.NumField() {
				field := param.Field(i)

				c, ok := cases[field.Name]
				if !ok {
					tb.Fatalf(
						"testo: wrong param signature for (%[1]s).%[2]s: missing (%[1]s).Cases%[3]s() []%s for param %[3]q",
						reflect.TypeFor[Suite](),
						method.Name,
						field.Name,
						field.Type,
					)
				}

				if !c.Provides.AssignableTo(field.Type) {
					//nolint:lll // splitting string into multiple lines is worse
					tb.Fatalf(
						"testo: wrong param signature for (%[1]s).%[2]s: (%[1]s).Cases%[3]s provides %[4]s values, not assignable to param %[3]q of type %[5]s",
						reflect.TypeFor[Suite](),
						method.Name,
						field.Name,
						c.Provides,
						field.Type,
					)
				}

				requiredCases[field.Name] = c
			}

			if !flagMethod.MatchString(method.Name) {
				continue
			}

			tests.Parametrized = append(
				tests.Parametrized,
				tc.newParametrizedTest(method, requiredCases),
			)
		}
	}

	return tests
}

type suiteTestParametrized[Suite suite[T], T CommonT] struct {
	Tests func(testing.TB, Suite) []annotatedSuiteTest[Suite, T]
}

func (tc *testsCollector[Suite, T]) newParametrizedTest(
	method reflect.Method,
	cases map[string]suiteCase[Suite, T],
) suiteTestParametrized[Suite, T] {
	return suiteTestParametrized[Suite, T]{
		Tests: func(tb testing.TB, s Suite) []annotatedSuiteTest[Suite, T] {
			tb.Helper()

			casesValues := make(map[string][]reflect.Value, len(cases))

			for caseName, c := range cases {
				values := c.Func(s)

				if len(values) == 0 {
					structName := method.Type.In(0).String()

					msg := fmt.Sprintf(
						"testo: (%[1]s).Cases%[2]s provides zero values, (%[1]s).%[3]s won't run",
						structName,
						caseName,
						method.Name,
					)

					if *flagStrict {
						tb.Fatal(msg)
					} else {
						tb.Log(msg)
					}

					return nil
				}

				casesValues[caseName] = values
			}

			permutations := casesPermutations(casesValues)

			tests := make([]annotatedSuiteTest[Suite, T], 0, len(permutations))

			for i, params := range permutations {
				paramValue := reflect.New(method.Type.In(2)).Elem()

				caseParams := make(map[string]any, len(params))

				for paramName, value := range params {
					paramValue.FieldByName(paramName).Set(value)

					caseParams[paramName] = value.Interface()
				}

				tests = append(tests, annotatedSuiteTest[Suite, T]{
					suiteTest: suiteTest[Suite, T]{
						Name: method.Name,
						Info: testoreflect.ParametrizedTestInfo{
							Name:       tc.testName(method.Name),
							BaseName:   method.Name,
							Index:      i,
							CasesCount: len(permutations),
							Params:     caseParams,
							FuncPC:     method.Func.Pointer(),
						},
						Run: func(s Suite, t T) {
							method.Func.Call([]reflect.Value{
								reflect.ValueOf(s),
								reflect.ValueOf(t),
								paramValue,
							})
						},
					},
					Options: annotationsFor(getID[Suite](method.Func)),
				})
			}

			return tests
		},
	}
}

// casesPermutations returns a determenistic permutations of the given cases values for test.
func casesPermutations[V any](v map[string][]V) []map[string]V {
	permutationsCount := 1
	keys := make([]string, 0, len(v))

	for key, values := range v {
		keys = append(keys, key)

		permutationsCount *= len(values)
	}

	// Sort keys for determenistic output
	slices.Sort(keys)

	permutations := make([]map[string]V, 0, permutationsCount)

	var generatePermutations func(current map[string]V, index int)

	generatePermutations = func(current map[string]V, index int) {
		if index == len(keys) {
			permutations = append(permutations, maps.Clone(current))

			return
		}

		key := keys[index]

		for _, val := range v[key] {
			current[key] = val
			generatePermutations(current, index+1)
		}
	}

	current := make(map[string]V)

	generatePermutations(current, 0)

	return permutations
}
