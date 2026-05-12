//go:build example

package main

import (
	"fmt"
	"slices"
	"time"

	"github.com/ozontech/testo"
	"github.com/ozontech/testo/testoplugin"
	"github.com/ozontech/testo/testoreflect"
)

type ReverseTestsOrder struct{}

// Plugins can implement this function to provide
// certain plugin functionality.
//
// It is optional - see AddNewMethods plugin.

func (ReverseTestsOrder) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
	return testoplugin.Spec{
		Plan: testoplugin.Plan{
			Prepare: func(_ testoreflect.SuiteInfo, tests *[]testoplugin.PlannedTest) {
				slices.Reverse(*tests)
			},
		},
	}
}

type OverrideLog struct{ *testo.T }

func (o OverrideLog) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
	return testoplugin.Spec{
		Overrides: testoplugin.Overrides{
			Log: func(f testoplugin.FuncLog) testoplugin.FuncLog {
				return func(args ...any) {
					o.Helper()

					// this will be printed each time t.Log is called.
					fmt.Println("Inside log override")
					f(args...)
				}
			},
		},
	}
}

// We can embed testo.T in plugins - it will be automatically initialized
// and share the same testo.T as an actual T from the current test.

type AddNewMethods struct{ *testo.T }

func (a AddNewMethods) Explode() { a.Fatal("BOOM") }

type Timer struct {
	*testo.T

	start time.Time
}

func (t *Timer) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
	return testoplugin.Spec{
		Hooks: testoplugin.Hooks{
			BeforeEach:    t.beforeEach(),
			BeforeEachSub: t.beforeEach(),
			AfterEachSub:  t.afterEach(),
			AfterEach:     t.afterEach(),
		},
	}
}

func (t *Timer) beforeEach() testoplugin.Hook {
	return testoplugin.Hook{
		Priority: testoplugin.TryFirst,
		Func: func() {
			// .Plugin() is called for each test, therefore
			// we can modify Timer fields safely (new instance for each test).
			t.start = time.Now()
		},
	}
}

func (t *Timer) afterEach() testoplugin.Hook {
	return testoplugin.Hook{
		Priority: testoplugin.TryFirst,
		Func: func() {
			elapsed := time.Since(t.start)

			fmt.Printf("Test %q took %s\n", t.Name(), elapsed)
		},
	}
}

type RunsInnerSubtest struct{ *testo.T }

func (r *RunsInnerSubtest) RunInnerSubtest() {
	testo.Run(r, "inner subtest which will trigger hooks", func(r *RunsInnerSubtest) {
		r.Log("Hi from inner subtest")
	})
}
