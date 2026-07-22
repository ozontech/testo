package testo

import (
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/ozontech/testo/internal/reflectutil"
	"github.com/ozontech/testo/testoplugin"
)

type testID string

var (
	globalAnnotationsMu sync.RWMutex
	globalAnnotations   = make(map[testID][]testoplugin.Option)
)

// ForEach applies static options (annotations) for each case of a given parametrized test.
//
// Similar to [For], but for parametrized tests.
//
//	testo.ForEach(MySuite.TestFoo, someplugin.WithThis(...), otherplugin.WithThat(...))
//	testo.ForEach((*Suite).TestFoo, myplugin.WithRetry())
func ForEach[Suite suite[T], T CommonT, P any](
	test func(Suite, T, P),
	options ...testoplugin.Option,
) struct{} {
	annotate(getID[Suite](reflect.ValueOf(test)), options...)

	return struct{}{}
}

// For applies static options (annotations) for a given test.
//
// Test annotations is a slice of options to be passed for this test.
// Test annotations are available to plugins before running an actual test,
// thus enhancing test planning features.
//
// Multiple annotation calls to the same test will append options.
//
//	testo.For(MySuite.TestFoo, someplugin.WithThis(...), otherplugin.WithThat(...))
//	testo.For((*Suite).TestFoo, myplugin.WithRetry())
//
// NOTE: it returns an empty struct{} value to enable the following usage:
//
//	var _ = testo.For(MySuite.TestFoo, someplugin.WithSomeOption(true))
//
//	func (MySuite) TestFoo(t T) { ... }
//
// That "var _ =" construction would not be possible otherwise.
// This is slightly less verbose than using an init function:
//
//	func init() {
//		testo.For(MySuite.TestFoo, someplugin.WithSomeOption(true))
//	}
func For[Suite suite[T], T CommonT](
	test func(Suite, T),
	options ...testoplugin.Option,
) struct{} {
	annotate(getID[Suite](reflect.ValueOf(test)), options...)

	return struct{}{}
}

func annotate(id testID, options ...testoplugin.Option) {
	globalAnnotationsMu.Lock()
	defer globalAnnotationsMu.Unlock()

	globalAnnotations[id] = append(globalAnnotations[id], options...)
}

func annotationsFor(id testID) []testoplugin.Option {
	globalAnnotationsMu.RLock()
	defer globalAnnotationsMu.RUnlock()

	return slices.Clone(globalAnnotations[id])
}

func getID[Suite any](test reflect.Value) testID {
	suiteName := reflectutil.NameOf[Suite]()

	name := runtime.FuncForPC(test.Pointer()).Name()

	name = strings.ReplaceAll(name, "(*"+suiteName+")", suiteName)

	// Runtime erases generic type arguments to "[...]" in function names:
	//
	//	KVSuite[int].TestGet    -> "pkg.KVSuite[...].TestGet"
	//	KVSuite[string].TestGet -> "pkg.KVSuite[...].TestGet" (same!)
	//
	// while suiteName keeps them ("KVSuite[int]").
	//
	// Code below gets us:
	//
	// 	KVSuite[int]|pkg.KVSuite[...].TestGet
	// 	KVSuite[string]|pkg.KVSuite[...].TestGet

	if base, _, isGeneric := strings.Cut(suiteName, "["); isGeneric {
		erased := base + "[...]"

		name = strings.ReplaceAll(name, "(*"+erased+")", erased)
	}

	return testID(suiteName + "|" + name)
}
