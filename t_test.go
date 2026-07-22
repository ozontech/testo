package testo

import (
	"context"
	"testing"
	"time"

	"github.com/ozontech/testo/testoplugin"
	"github.com/ozontech/testo/testoreflect"
)

type ReflectSuite struct {
	Suite[*T]

	beforeAll, beforeEach, test, afterEach, afterAll *testoreflect.Reflection
}

func (s *ReflectSuite) BeforeAll(t *T) {
	r := Reflect(t)

	s.beforeAll = &r
}

func (s *ReflectSuite) BeforeEach(t *T) {
	r := Reflect(t)

	s.beforeEach = &r
}

func (s *ReflectSuite) Test(t *T) {
	r := Reflect(t)

	s.test = &r
}

func (s *ReflectSuite) AfterEach(t *T) {
	r := Reflect(t)

	s.afterEach = &r
}

func (s *ReflectSuite) AfterAll(t *T) {
	r := Reflect(t)

	s.afterAll = &r
}

func TestReflect(t *testing.T) {
	t.Parallel()

	s := new(ReflectSuite)

	if !RunSuite(t, s) {
		t.Error("unexpected run suite failure")
	}

	type Case struct {
		Reflection *testoreflect.Reflection

		Level int
	}

	for name, c := range map[string]Case{
		"before all":  {Reflection: s.beforeAll, Level: 0},
		"before each": {Reflection: s.beforeEach, Level: 1},
		"test":        {Reflection: s.test, Level: 1},
		"after each":  {Reflection: s.afterEach, Level: 1},
		"after all":   {Reflection: s.afterAll, Level: 0},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if c.Reflection == nil {
				t.Fatal("nil reflection", name)
			}

			r := *c.Reflection

			if r.Test.GetLevel() != c.Level {
				t.Errorf("level not equal, want %d, got %d", c.Level, r.Test.GetLevel())
			}
		})
	}
}

type TSuiteT struct {
	*T
	*TSuitePlugin
}

type TSuitePlugin struct{}

type tSuiteCallCounts struct {
	log,
	parallel,
	setenv,
	tempDir,
	deadline,
	context,
	error,
	skip,
	skipNow,
	skipped,
	fail,
	failNow,
	failed,
	fatal int
	chdir   int
	cleanup int
}

var tSuiteOverridesCalls tSuiteCallCounts

func (p *TSuitePlugin) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
	return testoplugin.Spec{
		Overrides: testoplugin.Overrides{
			Log: func(f testoplugin.FuncLog) testoplugin.FuncLog {
				return func(args ...any) {
					tSuiteOverridesCalls.log++
				}
			},
			Parallel: func(f testoplugin.FuncParallel) testoplugin.FuncParallel {
				return func() {
					tSuiteOverridesCalls.parallel++
				}
			},
			Setenv: func(f testoplugin.FuncSetenv) testoplugin.FuncSetenv {
				return func(key, value string) {
					tSuiteOverridesCalls.setenv++
				}
			},
			TempDir: func(f testoplugin.FuncTempDir) testoplugin.FuncTempDir {
				return func() string {
					tSuiteOverridesCalls.tempDir++

					return ""
				}
			},
			Deadline: func(f testoplugin.FuncDeadline) testoplugin.FuncDeadline {
				return func() (deadline time.Time, ok bool) {
					tSuiteOverridesCalls.deadline++

					return deadline, ok
				}
			},
			Context: func(f testoplugin.FuncContext) testoplugin.FuncContext {
				return func() context.Context {
					tSuiteOverridesCalls.context++

					return context.Background()
				}
			},
			Error: func(f testoplugin.FuncError) testoplugin.FuncError {
				return func(args ...any) {
					tSuiteOverridesCalls.error++
				}
			},
			Skip: func(f testoplugin.FuncSkip) testoplugin.FuncSkip {
				return func(args ...any) {
					tSuiteOverridesCalls.skip++
				}
			},
			SkipNow: func(f testoplugin.FuncSkipNow) testoplugin.FuncSkipNow {
				return func() {
					tSuiteOverridesCalls.skipNow++
				}
			},
			Skipped: func(f testoplugin.FuncSkipped) testoplugin.FuncSkipped {
				return func() bool {
					tSuiteOverridesCalls.skipped++

					return false
				}
			},
			Fail: func(f testoplugin.FuncFail) testoplugin.FuncFail {
				return func() {
					tSuiteOverridesCalls.fail++
				}
			},
			FailNow: func(f testoplugin.FuncFailNow) testoplugin.FuncFailNow {
				return func() {
					tSuiteOverridesCalls.failNow++
				}
			},
			Failed: func(f testoplugin.FuncFailed) testoplugin.FuncFailed {
				return func() bool {
					tSuiteOverridesCalls.failed++

					return false
				}
			},
			Fatal: func(f testoplugin.FuncFatal) testoplugin.FuncFatal {
				return func(args ...any) {
					tSuiteOverridesCalls.fatal++
				}
			},
			Chdir: func(f testoplugin.FuncChdir) testoplugin.FuncChdir {
				return func(dir string) {
					tSuiteOverridesCalls.chdir++
				}
			},
			Cleanup: func(f testoplugin.FuncCleanup) testoplugin.FuncCleanup {
				return func(f func()) {
					tSuiteOverridesCalls.cleanup++
				}
			},
		},
	}
}

type TSuite struct {
	Suite[TSuiteT]
}

func (TSuite) Test(t TSuiteT) {
	t.Logf("")
	t.Log()
	t.Parallel()
	t.Setenv("", "")
	t.TempDir()
	t.Deadline()
	t.Context()
	t.Error()
	t.Errorf("")
	t.Skip()
	t.Skipf("")
	t.SkipNow()
	t.Skipped()
	t.Fail()
	t.FailNow()
	t.Failed()
	t.Fatal()
	t.Fatalf("")
	t.Cleanup(nil)
	t.Chdir("")
}

func TestSuiteT(t *testing.T) {
	// reset the counters so the test stays correct under -count>1
	tSuiteOverridesCalls = tSuiteCallCounts{}

	t.Parallel()

	if !RunSuite(t, new(TSuite)) {
		t.Fatal("unexpected suite failure")
	}

	type Case struct {
		Want, Got int
	}

	for name, c := range map[string]Case{
		"Log":      {2, tSuiteOverridesCalls.log},
		"Parallel": {1, tSuiteOverridesCalls.parallel},
		"Setenv":   {1, tSuiteOverridesCalls.setenv},
		"TempDir":  {1, tSuiteOverridesCalls.tempDir},
		"Deadline": {1, tSuiteOverridesCalls.deadline},
		"Context":  {1, tSuiteOverridesCalls.context},
		"Error":    {2, tSuiteOverridesCalls.error},
		"Skip":     {2, tSuiteOverridesCalls.skip},
		"SkipNow":  {1, tSuiteOverridesCalls.skipNow},
		"Skipped":  {3, tSuiteOverridesCalls.skipped}, // called by the framework twice for hooks
		"Fail":     {1, tSuiteOverridesCalls.fail},
		"FailNow":  {1, tSuiteOverridesCalls.failNow},
		"Failed":   {1, tSuiteOverridesCalls.failed},
		"Fatal":    {2, tSuiteOverridesCalls.fatal},
		"Chdir":    {1, tSuiteOverridesCalls.chdir},
		"Cleanup":  {1, tSuiteOverridesCalls.cleanup},
	} {
		if c.Got != c.Want {
			t.Errorf("%s: want calls(-s): %d, got %d", name, c.Want, c.Got)
		}
	}
}
