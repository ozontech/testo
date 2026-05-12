// Package testoreflect provides reflection primitives for T.
//
// You can obtain [Reflection] for any test by accessing its T instance:
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
package testoreflect

import (
	"fmt"
	"testing"
	"time"
)

// Reflection for T.
type Reflection struct {
	// Suite info which this test belongs to.
	Suite SuiteInfo

	// Test holds information about current test.
	Test TestInfo

	// Panic is panic information.
	// It is nil if the test did not panic.
	Panic *PanicInfo

	// FailureKind is test failure kind.
	FailureKind TestFailureKind

	// FailureSource is the test failure source.
	FailureSource TestFailureSource

	// HasFatalSubtest is true if any (nested) subtest of this test
	// has [TestFailureKind] equal to [TestFailureKindFatal].
	HasFatalSubtest bool

	// TestingT provides access to real [testing.T]
	// without overrides or hooks applied.
	TestingT TestingT
}

// TestingT is an interface for [testing.T].
type TestingT interface {
	testing.TB

	Deadline() (deadline time.Time, ok bool)
	Parallel()
	Run(name string, f func(t *testing.T)) bool
}

// TestFailureSource states where test failure appeared.
type TestFailureSource int

const (
	// TestFailureSourceNone states that test did not fail.
	TestFailureSourceNone TestFailureSource = iota

	// TestFailureSourceSelf states that test failed itself.
	TestFailureSourceSelf

	// TestFailureSourceChild states that test failed because of child failure.
	TestFailureSourceChild
)

// String implements [fmt.Stringer].
func (s TestFailureSource) String() string {
	switch s {
	case TestFailureSourceChild:
		return "child"

	case TestFailureSourceNone:
		return "none"

	case TestFailureSourceSelf:
		return "self"

	default:
		return fmt.Sprintf("TestFailureSource(%d)", s)
	}
}

// TestFailureKind defines test failure kinds.
type TestFailureKind int

const (
	// TestFailureKindNone states that test did not fail.
	TestFailureKindNone TestFailureKind = iota

	// TestFailureKindSoft states that test failed but it was not fatal.
	// For example, t.Fail() was called.
	TestFailureKindSoft

	// TestFailureKindFatal states that test failed with fatal error.
	// For example, t.FailNow() or t.Fatal() were called.
	TestFailureKindFatal
)

// String implements [fmt.Stringer].
func (k TestFailureKind) String() string {
	switch k {
	case TestFailureKindFatal:
		return "fatal"

	case TestFailureKindNone:
		return "none"

	case TestFailureKindSoft:
		return "soft"

	default:
		return fmt.Sprintf("TestFailureKind(%d)", k)
	}
}

// PanicInfo holds information for recovered panic.
type PanicInfo struct {
	// Value returned by recover().
	Value any

	// Trace is a stack trace for this panic.
	Trace string
}

// TestInfo is a enum which is
// either [ParametrizedTestInfo] or [RegularTestInfo].
//
//	switch info := info.(type) {
//	case testoreflect.ParametrizedTestInfo:
//	case testoreflect.RegularTestInfo:
//	}
type TestInfo interface {
	// GetName returns full test name as would be returned by t.Name().
	GetName() string

	// GetLevel returns test level (depth).
	//
	// 	func (Suite) BeforeAll(t T) {} // level 0
	//
	// 	func (Suite) BeforeEach(t T) {} // level 1
	//
	// 	func (Suite) Test(t T) { // level 1
	//		testo.Run(t, "...", func(t T) { // level 2
	//			testo.Run(t, "...", func(t T) { // level 3
	//			})
	//		})
	// 	}
	GetLevel() int

	isTestInfo()
}

// ParametrizedTestInfo is the information about parametrized test.
type ParametrizedTestInfo struct {
	// Name is a full test name as would be returned by t.Name().
	Name string

	// BaseName of the test.
	BaseName string

	// Index is the 0-based case number of this test.
	Index int

	// CasesCount is the total cases count for this test.
	CasesCount int

	// Params passed for the current test case.
	Params map[string]any

	// FuncPC is the program counter (PC) of this test function.
	//
	// NOTE: it may be empty (zero) in some cases, e.g. hooks and sub-tests.
	// Use with caution.
	FuncPC uintptr
}

// GetName returns value of [ParametrizedTestInfo.Name] field.
func (i ParametrizedTestInfo) GetName() string { return i.Name }

// GetLevel always returns level 1 since since parametrized tests can't be nested.
func (ParametrizedTestInfo) GetLevel() int { return 1 }

func (ParametrizedTestInfo) isTestInfo() {}

// SuiteInfo is the information about suite.
type SuiteInfo struct {
	// Name of this suite.
	Name string

	// Caller is the full test name from
	// which this suite was run.
	Caller string

	// Value holds suite value.
	Value any

	// Hooks information for this suite.
	Hooks SuiteHooksInfo
}

// SuiteHooksInfo is suite hooks information.
type SuiteHooksInfo struct {
	// MissedBeforeAll is true if suite did not call BeforeAll hook,
	// meaning it was not defined.
	MissedBeforeAll bool

	// MissedBeforeEach is true if suite did not call BeforeEach hook.
	// It is either not defined or failed beforehand.
	MissedBeforeEach bool

	// MissedAfterEach is true if suite did not call AfterEach hook.
	// It is either not defined or failed beforehand.
	MissedAfterEach bool

	// MissedBeforeAll is true if suite did not call BeforeAll hook,
	// meaning it was not defined.
	MissedAfterAll bool
}

// RegularTestInfo is the information about regular (non-parametrized) test.
type RegularTestInfo struct {
	// Name is a full test name as would be returned by t.Name().
	Name string

	// RawBaseName is the raw "unformatted" base name of this test.
	//
	// For example:
	//
	//   Run(t, "my test name!?", func(...) { ... })
	//
	// t.Name() would equal to "SomeSuite/my_test_name",
	// while this field would equal to "my test name!?" (the same as passed).
	RawBaseName string

	// Level indicates how deep this t is.
	// That is, it shows the number of parents it has and zero if none.
	//
	// Zero level is Before/After-All hooks.
	// Subsequent levels are for test methods or subtests ran from Before/After-All hooks.
	//
	// To differentiate subtest ran from Before/After-All hook from a test method see IsSubtest field.
	Level int

	// IsSubtest states if this test is a subtest.
	//
	// Example of a subtest:
	// 	testo.Run(t, "...", func(T) {})
	//
	// Example of a test method (not subtest):
	// 	func (Suite) TestFoo(T)
	IsSubtest bool

	// FuncPC is the program counter (PC) of this test function.
	//
	// NOTE: it may be empty (zero) in some cases, e.g. hooks and sub-tests.
	// Use with caution.
	FuncPC uintptr
}

// GetName returns value of [RegularTestInfo.Name] field.
func (i RegularTestInfo) GetName() string { return i.Name }

// GetLevel returns value of [RegularTestInfo.Level] field.
func (i RegularTestInfo) GetLevel() int { return i.Level }

func (RegularTestInfo) isTestInfo() {}
