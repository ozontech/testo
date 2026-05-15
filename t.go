//nolint:funcorder // private and public methods close to each other for readability
package testo

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ozontech/testo/internal/reflectutil"
	"github.com/ozontech/testo/internal/testnamer"
	"github.com/ozontech/testo/testoplugin"
	"github.com/ozontech/testo/testoreflect"
)

type common interface {
	testing.TB

	Deadline() (deadline time.Time, ok bool)
	Parallel()
}

// TestingT is an interface for [testing.T].
type TestingT interface {
	common

	Run(name string, f func(t *testing.T)) bool
}

// CommonT is the interface common for all [T] derivatives.
type CommonT interface {
	common

	unwrap() *T
}

type (
	// T is a wrapper for [testing.T].
	// This is a core entity in testo and used as a [testing.T] replacement.
	//
	// The common pattern is to embed it into new struct type:
	//
	//  type MyT struct {
	//  	*testo.T
	//  	*SomePlugin
	//  }
	//
	// Plugins can also optionally embed it - testo will automatically initialize it
	// by sharing the same value as an actual currently running test's T.
	//
	//  type SomePlugin struct { *testo.T }
	T struct {
		common

		// we double testing t interfaces,
		// so that we still can have access for testing.T.Run
		// but the user don't.
		testingT TestingT

		testNamer *testnamer.Namer

		parent *T
		spec   testoplugin.Spec

		// levelOptions stores options passed for the
		// current level through [Run], [RunSuite] or [For].
		levelOptions []testoplugin.Option

		// reflection holds information for [Reflect].
		reflection testoreflect.Reflection

		failureSource   atomicInt[testoreflect.TestFailureSource]
		failureKind     atomicInt[testoreflect.TestFailureKind]
		hasFatalSubtest atomic.Bool

		plugins map[reflect.Type]testoplugin.Plugin
	}

	testoT = T
)

// Plugin implements [testoplugin.Plugin].
//
// This is a placeholder to prevent other .Plugin methods being promoted.
func (*T) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
	return testoplugin.Spec{}
}

// Context returns a context that is canceled just before
// Cleanup-registered functions are called.
//
// Cleanup functions can wait for any resources
// that shut down on [context.Context.Done] before the test completes.
func (t *T) Context() context.Context {
	t.Helper()

	return t.spec.Overrides.Context.Call(t.common.Context)()
}

// Parallel signals that this test is to be run in parallel with (and only with)
// other parallel tests. When a test is run multiple times due to use of
// -test.count or -test.cpu, multiple instances of a single test never run in
// parallel with each other.
func (t *T) Parallel() {
	t.Helper()

	t.spec.Overrides.Parallel.Call(t.common.Parallel)()
}

// Setenv calls os.Setenv(key, value) and uses Cleanup to
// restore the environment variable to its original value
// after the test.
//
// Because Setenv affects the whole process, it cannot be used
// in parallel tests or tests with parallel ancestors.
func (t *T) Setenv(key, value string) {
	t.Helper()

	t.spec.Overrides.Setenv.Call(t.common.Setenv)(key, value)
}

// TempDir returns a temporary directory for the test to use.
// The directory is automatically removed when the test and
// all its subtests complete.
// Each subsequent call to t.TempDir returns a unique directory;
// if the directory creation fails, TempDir terminates the test by calling Fatal.
func (t *T) TempDir() string {
	t.Helper()

	return t.spec.Overrides.TempDir.Call(t.common.TempDir)()
}

// Log formats its arguments using default formatting, analogous to Println,
// and records the text in the error log. For tests, the text will be printed only if
// the test fails or the -test.v flag is set. For benchmarks, the text is always
// printed to avoid having performance depend on the value of the -test.v flag.
func (t *T) Log(args ...any) {
	t.Helper()

	t.spec.Overrides.Log.Call(t.common.Log)(args...)
}

// Logf formats its arguments according to the format, analogous to Printf, and
// records the text in the error log. A final newline is added if not provided. For
// tests, the text will be printed only if the test fails or the -test.v flag is
// set. For benchmarks, the text is always printed to avoid having performance
// depend on the value of the -test.v flag.
func (t *T) Logf(format string, args ...any) {
	t.Helper()

	t.Log(fmt.Sprintf(format, args...))
}

// Deadline reports the time at which the test binary will have
// exceeded the timeout specified by the -timeout flag.
//
// By default, the ok result is false if the -timeout flag indicates "no timeout" (0).
func (t *T) Deadline() (time.Time, bool) {
	t.Helper()

	return t.spec.Overrides.Deadline.Call(t.common.Deadline)()
}

// Errorf is equivalent to Error with formatted message.
func (t *T) Errorf(format string, args ...any) {
	t.Helper()

	t.Error(fmt.Sprintf(format, args...))
}

// Error is equivalent to Log followed by Fail.
func (t *T) Error(args ...any) {
	t.Helper()

	t.spec.Overrides.Error.Call(t.error)(args...)
}

func (t *T) error(args ...any) {
	t.Helper()

	t.Log(args...)
	t.Fail()
}

// Skip is equivalent to Log followed by SkipNow.
func (t *T) Skip(args ...any) {
	t.Helper()

	t.spec.Overrides.Skip.Call(t.skip)(args...)
}

func (t *T) skip(args ...any) {
	t.Helper()

	t.Log(args...)
	t.SkipNow()
}

// SkipNow marks the test as having been skipped and stops its execution
// by calling [runtime.Goexit].
// If a test fails (see Error, Errorf, Fail) and is then skipped,
// it is still considered to have failed.
// Execution will continue at the next test or benchmark. See also FailNow.
// SkipNow must be called from the goroutine running the test, not from
// other goroutines created during the test. Calling SkipNow does not stop
// those other goroutines.
func (t *T) SkipNow() {
	t.Helper()

	t.spec.Overrides.SkipNow.Call(t.common.SkipNow)()
}

// Skipf is equivalent to Skip with formatted message.
func (t *T) Skipf(format string, args ...any) {
	t.Helper()

	t.Skip(fmt.Sprintf(format, args...))
}

// Skipped reports whether the test was skipped.
func (t *T) Skipped() bool {
	t.Helper()

	return t.spec.Overrides.Skipped.Call(t.common.Skipped)()
}

// Fail marks the function as having failed but continues execution.
func (t *T) Fail() {
	t.Helper()

	t.spec.Overrides.Fail.Call(t.fail)()
}

func (t *T) fail() {
	t.Helper()

	t.markFailure(testoreflect.TestFailureKindSoft)
	t.common.Fail()
}

// FailNow marks the function as having failed and stops its execution
// by calling runtime.Goexit (which then runs all deferred calls in the
// current goroutine).
// Execution will continue at the next test or benchmark.
// FailNow must be called from the goroutine running the
// test or benchmark function, not from other goroutines
// created during the test. Calling FailNow does not stop
// those other goroutines.
func (t *T) FailNow() {
	t.Helper()

	t.spec.Overrides.FailNow.Call(t.failNow)()
}

func (t *T) failNow() {
	t.Helper()

	t.markFailure(testoreflect.TestFailureKindFatal)
	t.common.FailNow()
}

// Failed reports whether the function has failed.
func (t *T) Failed() bool {
	t.Helper()

	return t.spec.Overrides.Failed.Call(t.common.Failed)()
}

// Fatal is equivalent to Log followed by FailNow.
func (t *T) Fatal(args ...any) {
	t.Helper()

	t.spec.Overrides.Fatal.Call(t.fatal)(args...)
}

func (t *T) fatal(args ...any) {
	t.Helper()

	t.Log(args...)
	t.FailNow()
}

// Fatalf is equivalent to Fatal with formatted message.
func (t *T) Fatalf(format string, args ...any) {
	t.Helper()

	t.Fatal(fmt.Sprintf(format, args...))
}

// Name returns the name of the running (sub-) test or benchmark.
//
// The name will include the name of the test along with the names of
// any nested sub-tests. If two sibling sub-tests have the same name,
// Name will append a suffix to guarantee the returned name is unique.
func (t *T) Name() string {
	t.Helper()

	return t.reflection.Test.GetName()
}

// unwrap the underlying T.
//
// It works since T's are embedded in user-defined structs.
func (t *T) unwrap() *T {
	return t
}

// level indicates how deep this t is.
// That is, it shows the number of parents it has and zero if none.
func (t *T) level() int {
	var level int

	parent := t.parent

	for parent != nil {
		level++

		parent = parent.parent
	}

	return level
}

// markFailure sets failure kind to the current t
// and promotes it for all ancestors.
func (t *T) markFailure(kind testoreflect.TestFailureKind) {
	if kind == testoreflect.TestFailureKindNone {
		return
	}

	t.failureKind.Store(kind)
	t.failureSource.Store(testoreflect.TestFailureSourceSelf)

	parent := t.parent

	for parent != nil {
		// parent may already have fatal failure,
		// so we overwrite parent failure kind only if it has none.
		parent.failureKind.CompareAndSwap(
			testoreflect.TestFailureKindNone,
			testoreflect.TestFailureKindSoft,
		)

		parent.failureSource.CompareAndSwap(
			testoreflect.TestFailureSourceNone,
			testoreflect.TestFailureSourceChild,
		)

		if kind == testoreflect.TestFailureKindFatal {
			parent.hasFatalSubtest.Store(true)
		}

		parent = parent.parent
	}
}

func (t *T) options() []testoplugin.Option {
	size := len(t.levelOptions)
	byLevel := [][]testoplugin.Option{t.levelOptions}

	parent := t.parent

	for parent != nil {
		level := make([]testoplugin.Option, 0, len(parent.levelOptions))

		for _, o := range parent.levelOptions {
			if o.Propagate {
				level = append(level, o)
			}
		}

		byLevel = append(byLevel, level)

		size += len(level)

		parent = parent.parent
	}

	options := make([]testoplugin.Option, 0, size)

	// so that child options come after parent options

	for _, level := range slices.Backward(byLevel) {
		options = append(options, level...)
	}

	return options
}

func (t *T) pluginNames() []string {
	names := make([]string, 0, len(t.unwrap().plugins))

	for typ := range t.unwrap().plugins {
		if typ == reflect.TypeFor[*T]() {
			continue
		}

		names = append(names, reflectutil.Elem(typ).String())
	}

	slices.Sort(names)

	return names
}

func (t *T) logPlugins() {
	t.Helper()

	names := t.unwrap().pluginNames()

	noun := "plugins"
	if len(names) == 1 {
		noun = "plugin"
	}

	t.testingT.Logf(
		"testo: collected %d %s: %s\n",
		len(names),
		noun,
		strings.Join(names, ", "),
	)
}

// Reflect returns meta information about given t.
//
// You can reflect over any test by accessing its T instance:
//
//	func (Suite) TestFoo(t T) {
//		r := testo.Reflect(t)
//		// r stores Reflection struct.
//	}
//
// Same logic applies for plugins.
// If a plugin embeds `*testo.T` it can call the same [testo.Reflect] function:
//
//	type Plugin struct{ *testo.T }
//
//	func (p *Plugin) Plugin(parent testoplugin.Plugin, options ...testoplugin.Options) testoplugin.Spec {
//		return testoplugin.Spec{
//			Hooks: testoplugin.Hooks{
//	 			BeforeEach: testoplugin.Hook{
//	 				Func: func() { testo.Reflect(p) }
//				}
//			}
//		 }
//	}
func Reflect(t CommonT) testoreflect.Reflection {
	t.Helper()

	internal := t.unwrap()

	info := internal.reflection

	info.FailureSource = internal.failureSource.Load()
	info.FailureKind = internal.failureKind.Load()
	info.HasFatalSubtest = internal.hasFatalSubtest.Load()
	info.TestingT = internal.testingT

	return info
}

type atomicInt[T ~int | ~int8 | ~int32 | ~int64] atomic.Int64

func (a *atomicInt[T]) Load() T {
	return T((*atomic.Int64)(a).Load())
}

func (a *atomicInt[T]) Store(value T) {
	(*atomic.Int64)(a).Store(int64(value))
}

func (a *atomicInt[T]) CompareAndSwap(oldvalue, newvalue T) bool {
	return (*atomic.Int64)(a).CompareAndSwap(int64(oldvalue), int64(newvalue))
}
