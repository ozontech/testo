package testo

import (
	"cmp"
	"slices"
	"testing"

	"github.com/ozontech/testo/testoplugin"
	"github.com/ozontech/testo/testoreflect"
)

// mergeSpecs multiple plugin specs into one.
func mergeSpecs(tb testing.TB, plugins ...testoplugin.Spec) testoplugin.Spec {
	tb.Helper()

	plans := make([]testoplugin.Plan, 0, len(plugins))
	hooks := make([]testoplugin.Hooks, 0, len(plugins))
	overrides := make([]testoplugin.Overrides, 0, len(plugins))

	for _, p := range plugins {
		plans = append(plans, p.Plan)
		hooks = append(hooks, p.Hooks)
		overrides = append(overrides, p.Overrides)
	}

	return testoplugin.Spec{
		Plan:      mergePlans(tb, plans...),
		Hooks:     mergeHooks(tb, hooks...),
		Overrides: mergeOverrides(overrides...),
	}
}

func mergePlans(tb testing.TB, plans ...testoplugin.Plan) testoplugin.Plan {
	tb.Helper()

	return testoplugin.Plan{
		Prepare: func(suite testoreflect.SuiteInfo, tests *[]testoplugin.PlannedTest) {
			tb.Helper()

			// We could've break the loop when len(tests) == 0
			// but it may be useful if some plugin would want to throw some warning or error
			// when len(tests) == 0. Something like NoEmptySuitesPlugin.
			for _, p := range plans {
				if p.Prepare != nil {
					p.Prepare(suite, tests)
				}
			}
		},
	}
}

func mergeHooks(tb testing.TB, hooks ...testoplugin.Hooks) testoplugin.Hooks {
	tb.Helper()

	beforeAll := make([]testoplugin.Hook, 0, len(hooks))
	beforeEach := make([]testoplugin.Hook, 0, len(hooks))
	beforeEachSub := make([]testoplugin.Hook, 0, len(hooks))
	afterEachSub := make([]testoplugin.Hook, 0, len(hooks))
	afterEach := make([]testoplugin.Hook, 0, len(hooks))
	afterAll := make([]testoplugin.Hook, 0, len(hooks))

	for _, h := range hooks {
		if h := h.BeforeAll; h.Func != nil {
			beforeAll = append(beforeAll, h)
		}

		if h := h.BeforeEach; h.Func != nil {
			beforeEach = append(beforeEach, h)
		}

		if h := h.BeforeEachSub; h.Func != nil {
			beforeEachSub = append(beforeEachSub, h)
		}

		if h := h.AfterEachSub; h.Func != nil {
			afterEachSub = append(afterEachSub, h)
		}

		if h := h.AfterEach; h.Func != nil {
			afterEach = append(afterEach, h)
		}

		if h := h.AfterAll; h.Func != nil {
			afterAll = append(afterAll, h)
		}
	}

	run := func(hooks []testoplugin.Hook) func() {
		tb.Helper()

		slices.SortStableFunc(hooks, func(a, b testoplugin.Hook) int {
			return cmp.Compare(a.Priority, b.Priority)
		})

		return func() {
			tb.Helper()

			for _, h := range hooks {
				runHook(tb, h)
			}
		}
	}

	return testoplugin.Hooks{
		BeforeAll:     testoplugin.Hook{Func: run(beforeAll)},
		BeforeEach:    testoplugin.Hook{Func: run(beforeEach)},
		BeforeEachSub: testoplugin.Hook{Func: run(beforeEachSub)},
		AfterEachSub:  testoplugin.Hook{Func: run(afterEachSub)},
		AfterEach:     testoplugin.Hook{Func: run(afterEach)},
		AfterAll:      testoplugin.Hook{Func: run(afterAll)},
	}
}

//nolint:funlen // splitting this into subfunctons would make it worse
func mergeOverrides(overrides ...testoplugin.Overrides) testoplugin.Overrides {
	return testoplugin.Overrides{
		Log: mergeOverride(
			overrides,
			func(o testoplugin.Overrides) testoplugin.Override[testoplugin.FuncLog] {
				return o.Log
			},
		),
		Parallel: mergeOverride(
			overrides,
			func(o testoplugin.Overrides) testoplugin.Override[testoplugin.FuncParallel] {
				return o.Parallel
			},
		),
		Setenv: mergeOverride(
			overrides,
			func(o testoplugin.Overrides) testoplugin.Override[testoplugin.FuncSetenv] {
				return o.Setenv
			},
		),
		TempDir: mergeOverride(
			overrides,
			func(o testoplugin.Overrides) testoplugin.Override[testoplugin.FuncTempDir] {
				return o.TempDir
			},
		),
		Deadline: mergeOverride(
			overrides,
			func(o testoplugin.Overrides) testoplugin.Override[testoplugin.FuncDeadline] {
				return o.Deadline
			},
		),
		Context: mergeOverride(
			overrides,
			func(o testoplugin.Overrides) testoplugin.Override[testoplugin.FuncContext] {
				return o.Context
			},
		),
		Error: mergeOverride(
			overrides,
			func(o testoplugin.Overrides) testoplugin.Override[testoplugin.FuncError] {
				return o.Error
			},
		),
		Skip: mergeOverride(
			overrides,
			func(o testoplugin.Overrides) testoplugin.Override[testoplugin.FuncSkip] {
				return o.Skip
			},
		),
		SkipNow: mergeOverride(
			overrides,
			func(o testoplugin.Overrides) testoplugin.Override[testoplugin.FuncSkipNow] {
				return o.SkipNow
			},
		),
		Skipped: mergeOverride(
			overrides,
			func(o testoplugin.Overrides) testoplugin.Override[testoplugin.FuncSkipped] {
				return o.Skipped
			},
		),
		Fail: mergeOverride(
			overrides,
			func(o testoplugin.Overrides) testoplugin.Override[testoplugin.FuncFail] {
				return o.Fail
			},
		),
		FailNow: mergeOverride(
			overrides,
			func(o testoplugin.Overrides) testoplugin.Override[testoplugin.FuncFailNow] {
				return o.FailNow
			},
		),
		Failed: mergeOverride(
			overrides,
			func(o testoplugin.Overrides) testoplugin.Override[testoplugin.FuncFailed] {
				return o.Failed
			},
		),
		Fatal: mergeOverride(
			overrides,
			func(o testoplugin.Overrides) testoplugin.Override[testoplugin.FuncFatal] {
				return o.Fatal
			},
		),
	}
}

func mergeOverride[F any](
	overrides []testoplugin.Overrides,
	getter func(testoplugin.Overrides) testoplugin.Override[F],
) func(F) F {
	return func(f F) F {
		for _, o := range overrides {
			if override := getter(o); override != nil {
				f = override(f)
			}
		}

		return f
	}
}
