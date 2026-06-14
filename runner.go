package testo

import (
	"fmt"
	"path"
	"reflect"
	"runtime/debug"
	"testing"

	"github.com/ozontech/testo/internal/reflectutil"
	"github.com/ozontech/testo/internal/testnamer"
	"github.com/ozontech/testo/testoplugin"
	"github.com/ozontech/testo/testoreflect"
)

// parallelWrapperTest is the name of tests which
// wrap multiple (possibly parallel) tests to ensure
// hooks are executed properly.
//
// It should contain some special symbol which identifiers in Go
// cannot include (like exclamation mark), so that it won't collide with suite type name.
const parallelWrapperTest = "testo!"

// Test constructs a new test ready to run as a native [testing] test.
//
// # Examples
//
//	func Test(t *testing.T) {
//		t.Run("My awesome test", testo.Test(func(t T) {
//			// your test goes here
//		}))
//	}
//
// This is syntactic sugar for a more verbose [RunTest] API:
//
//	func Test(t *testing.T) {
//		t.Run("My awesome test", func(t *testing.T) {
//			testo.RunTest(t, func(t T) {
//				// your test goes here
//			})
//		})
//	}
//
// # Options
//
// This function accepts plugin options, see [testoplugin.Option].
// Passed options are treated as test scoped, not suite scoped.
func Test[T CommonT](f func(t T), options ...testoplugin.Option) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()

		RunTest(t, f, options...)
	}
}

// RunTest runs a single test without a suite.
//
// Under the hood it constructs a special singleton suite with one test, named
// as the parent test, and calls [RunSuite].
//
// # Examples
//
//	func TestFoo(t *testing.T) {
//		testo.RunTest(t, func(t T) {
//			t.Log("Hi")
//		})
//	}
//
// In the example above plugins would see this test as a suite with a single TestFoo method.
//
// See also [Test] as a syntax sugar to run a named test:
//
//	func TestFoo(t *testing.T) {
//		t.Run("named-test", testo.Test(func(t T) {
//			t.Log("Hi")
//		}))
//	}
//
// # Options
//
// This function accepts plugin options, see [testoplugin.Option].
// Passed options are treated as test scoped, not suite scoped.
//
// # Note
//
// Running this function more than once inside the same test
// means rerunning the same test, not running several different tests.
// If you want to run several suite-less tests from a single test see [Test].
//
// RunTest reports whether f succeeded.
func RunTest[T CommonT](
	testingT TestingT,
	f func(t T),
	options ...testoplugin.Option,
) bool {
	testingT.Helper()

	s := singleton[T]{
		test:    f,
		name:    path.Base(testingT.Name()),
		options: options,
	}

	return RunSuite(testingT, s)
}

// RunSuite runs tests under a suite.
//
// Test is defined as a suite method in the form of "TestXXX" or "Test"
// which accepts a single parameter of the same type as T passed to this function.
//
// It also accepts options for the plugins which can be used to configure those plugins.
// See [testoplugin.Option].
//
// RunSuite reports whether suite succeeded.
func RunSuite[Suite suite[T], T CommonT](
	testingT TestingT,
	suite Suite,
	options ...testoplugin.Option,
) bool {
	testingT.Helper()

	r := newRunner[Suite](testingT)

	return r.runSuite(testingT, suite, nil, options...)
}

// RunSubSuite runs a sub-suite.
//
// This is similar to [RunSuite] but designed to be called from other suites.
//
// RunSubSuite reports whether all sub-suite tests succeeded.
//
// NOTE: this function may cause infinite loop if called within the same suite as passed to it.
func RunSubSuite[Suite suite[Sub], Parent, Sub CommonT](
	t Parent,
	suite Suite,
	options ...testoplugin.Option,
) bool {
	t.Helper()

	r := newRunner[Suite](t)

	parent := t.unwrap().reflection.Load().Suite

	return r.runSuite(t.unwrap().testingT, suite, &parent, options...)
}

// Run runs f as a subtest of t called name. It runs f in a separate goroutine
// and blocks until f returns or calls t.Parallel to become a parallel test.
// Run reports whether f succeeded (or at least did not fail before calling t.Parallel).
//
// Run may be called simultaneously from multiple goroutines, but all such calls
// must return before the outer test function for t returns.
//
// WARN: Running this function during t.Cleanup panics.
func Run[T CommonT](
	t T,
	name string,
	f func(t T),
	options ...testoplugin.Option,
) bool {
	t.Helper()

	if f == nil {
		f = func(T) {}
	}

	parentT := t

	return parentT.unwrap().testingT.Run(name, func(testingT *testing.T) {
		testingT.Helper()

		t := construct(
			testingT,
			&parentT,
			func(t *testoT) {
				t.testNamer = parentT.unwrap().testNamer

				parentSuite := parentT.unwrap().reflection.Load().Suite

				t.reflection.Modify(func(r *testoreflect.Reflection) {
					r.Suite = parentSuite
					r.Test = testoreflect.RegularTestInfo{
						Name:        parentT.unwrap().testNamer.Name(parentT.unwrap().Name(), name),
						RawBaseName: name,
						Level:       t.level(),
						IsSubtest:   true,
						FuncPC:      reflect.ValueOf(f).Pointer(),
					}
				})
			},
			options...,
		)

		defer func() {
			if r := recover(); r != nil {
				trace := string(debug.Stack())

				t.unwrap().reflection.Modify(func(r *testoreflect.Reflection) {
					r.Panic = &testoreflect.PanicInfo{
						Value: r,
						Trace: trace,
					}
				})

				t.Fatalf("testo: test %q panicked: %v\n\n%s", t.Name(), r, trace)
			}
		}()

		defer runHook(t, t.unwrap().spec.Hooks.AfterEachSub)

		runHook(t, t.unwrap().spec.Hooks.BeforeEachSub)

		f(t)
	})
}

type runner[Suite suite[T], T CommonT] struct {
	caller    string
	suiteName string
	testNamer *testnamer.Namer
}

func newRunner[Suite suite[T], T CommonT](t common) runner[Suite, T] {
	suiteName := reflectutil.NameOf[Suite]()
	if suiteName == reflectutil.NameOf[singleton[T]]() {
		suiteName = ""
	}

	namer := testnamer.New()

	return runner[Suite, T]{
		caller:    namer.Name(t.Name(), suiteName),
		suiteName: suiteName,
		testNamer: namer,
	}
}

func (r *runner[Suite, T]) collectTests(
	t TestingT,
) suiteTests[Suite, T] {
	t.Helper()

	collector := testsCollector[Suite, T]{
		CallerName: r.caller,
		TestNamer:  r.testNamer,
	}

	return collector.Collect(t)
}

func (r *runner[Suite, T]) runSuite(
	testingT TestingT,
	suite Suite,
	parentSuite *testoreflect.SuiteInfo,
	options ...testoplugin.Option,
) bool {
	testingT.Helper()

	options = append(getOptions(), options...)

	suiteInfo := testoreflect.SuiteInfo{
		Parent:   parentSuite,
		Name:     r.suiteName,
		Caller:   testingT.Name(),
		TestingT: testingT,
		Value:    suite,
	}

	return testingT.Run(r.suiteName, func(testingT *testing.T) {
		testingT.Helper()

		tests := r.collectTests(testingT)

		t := construct[T](
			testingT,
			nil,
			func(t *testoT) {
				t.testNamer = r.testNamer

				t.reflection.Modify(func(ref *testoreflect.Reflection) {
					ref.Suite = suiteInfo
					ref.Test = testoreflect.RegularTestInfo{
						Name:        r.caller,
						RawBaseName: r.suiteName,
					}
				})
			},
			options...,
		)

		t.unwrap().logPlugins()

		r.runSuiteTests(t, suite, tests)
	})
}

func runHook(t testing.TB, h testoplugin.Hook) {
	t.Helper()

	if h.Func != nil {
		h.Func()
	}
}

func (r *runner[Suite, T]) runSuiteTests(t T, s Suite, tests suiteTests[Suite, T]) {
	t.Helper()

	defer func() {
		if !t.Skipped() {
			runHook(t, t.unwrap().spec.Hooks.AfterAll)
		}
	}()

	runHook(t, t.unwrap().spec.Hooks.BeforeAll)

	defer func() {
		if !t.Skipped() {
			s.AfterAll(t)
		}
	}()

	s.BeforeAll(t)

	suiteReflection := t.unwrap().reflection.Load().Suite

	suiteInfo := testoreflect.SuiteInfo{
		Parent:   suiteReflection.Parent,
		Name:     suiteReflection.Name,
		Caller:   suiteReflection.Caller,
		TestingT: suiteReflection.TestingT,
		Value:    s,
		Hooks:    suiteReflection.Hooks,
	}

	allTests := r.applyPlan(
		t,
		suiteInfo,
		tests.Collect(t, s, func(name string) string {
			return r.testNamer.Name(r.caller, name)
		}),
	)

	t.unwrap().testingT.Run(parallelWrapperTest, func(testingT *testing.T) {
		testingT.Helper()

		for _, test := range allTests {
			testingT.Run(test.Name, func(testingT *testing.T) {
				testingT.Helper()

				innerT := construct(
					testingT,
					&t,
					func(t *testoT) {
						t.testNamer = r.testNamer

						t.reflection.Modify(func(r *testoreflect.Reflection) {
							r.Suite = suiteInfo
							r.Test = test.Info
						})

						if test.Configure != nil {
							test.Configure(t)
						}
					},
					test.Options...,
				)

				r.runSuiteTest(
					innerT,
					s,
					test.suiteTest,
				)
			})
		}
	})
}

func (r *runner[Suite, T]) runSuiteTest(
	t T,
	s Suite,
	test suiteTest[Suite, T],
) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			trace := string(debug.Stack())

			t.unwrap().reflection.Modify(func(ref *testoreflect.Reflection) {
				ref.Panic = &testoreflect.PanicInfo{
					Value: r,
					Trace: trace,
				}
			})

			t.Fatalf("testo: test %q panicked: %v\n\n%s", t.Name(), r, trace)
		}
	}()

	defer runHook(t, t.unwrap().spec.Hooks.AfterEach)

	runHook(t, t.unwrap().spec.Hooks.BeforeEach)

	defer s.AfterEach(t)

	s.BeforeEach(t)

	test.Run(s, t)
}

func (r *runner[Suite, T]) applyPlan(
	t T,
	suiteInfo testoreflect.SuiteInfo,
	tests []annotatedSuiteTest[Suite, T],
) []annotatedSuiteTest[Suite, T] {
	t.Helper()

	plannedTests := make([]testoplugin.PlannedTest, 0, len(tests))

	for _, t := range tests {
		plannedTests = append(plannedTests, plannedSuiteTest[Suite, T]{t})
	}

	if prepare := t.unwrap().spec.Plan.Prepare; prepare != nil {
		prepare(suiteInfo, &plannedTests)
	}

	testsToReturn := make([]annotatedSuiteTest[Suite, T], 0, len(plannedTests))

	for _, t := range plannedTests {
		if t == nil {
			continue
		}

		planned, ok := t.(plannedSuiteTest[Suite, T])
		if !ok {
			// must be unreachable because of "DoNotImplement" directive.
			panic(fmt.Sprintf(
				"testo: planned test is not of type %q",
				reflect.TypeFor[plannedSuiteTest[Suite, T]](),
			))
		}

		testsToReturn = append(testsToReturn, planned.inner)
	}

	return testsToReturn
}
