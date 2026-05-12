package testo

import (
	"slices"
	"testing"

	"github.com/ozontech/testo/testoplugin"
)

type TestT struct {
	*T

	*TestPlugin
}

type TestSuite struct {
	Suite[TestT]

	beforeAllTriggered bool
}

type TestPlugin struct{ *T }

func (t *TestPlugin) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
	return testoplugin.Spec{
		Hooks: testoplugin.Hooks{
			BeforeAll: testoplugin.Hook{
				Func: func() {
					pluginBeforeAll = append(pluginBeforeAll, t.Name())
				},
			},
			BeforeEach: testoplugin.Hook{
				Func: func() {
					pluginBeforeEach = append(pluginBeforeEach, t.Name())
				},
			},
			BeforeEachSub: testoplugin.Hook{
				Func: func() {
					pluginBeforeEachSub = append(pluginBeforeEachSub, t.Name())
				},
			},
			AfterEachSub: testoplugin.Hook{
				Func: func() {
					pluginAfterEachSub = append(pluginAfterEachSub, t.Name())
				},
			},
			AfterEach: testoplugin.Hook{
				Func: func() {
					pluginAfterEach = append(pluginAfterEach, t.Name())
				},
			},
			AfterAll: testoplugin.Hook{
				Func: func() {
					pluginAfterAll = append(pluginAfterAll, t.Name())
				},
			},
		},
	}
}

var (
	beforeAll  []string
	beforeEach []string
	afterEach  []string
	afterAll   []string

	pluginBeforeAll     []string
	pluginBeforeEach    []string
	pluginBeforeEachSub []string
	pluginAfterEachSub  []string
	pluginAfterEach     []string
	pluginAfterAll      []string
)

func TestRunSuite(t *testing.T) {
	t.Parallel()

	beforeAll = nil
	beforeEach = nil
	afterEach = nil
	afterAll = nil

	pluginBeforeAll = nil
	pluginBeforeEach = nil
	pluginBeforeEachSub = nil
	pluginAfterEachSub = nil
	pluginAfterEach = nil
	pluginAfterAll = nil

	RunSuite(t, new(TestSuite))

	equal := func(want, got []string) {
		t.Helper()

		if !slices.Equal(want, got) {
			t.Errorf("not equal: want %+v, got %+v", want, got)
		}
	}

	equal([]string{"TestRunSuite/TestSuite"}, beforeAll)
	equal([]string{"TestRunSuite/TestSuite"}, pluginBeforeAll)

	equal([]string{
		"TestRunSuite/TestSuite/TestBar",
		"TestRunSuite/TestSuite/TestFoo",
	}, beforeEach)
	equal([]string{
		"TestRunSuite/TestSuite/TestBar",
		"TestRunSuite/TestSuite/TestFoo",
	}, pluginBeforeEach)

	equal([]string{"TestRunSuite/TestSuite/TestFoo/subtest"}, pluginBeforeEachSub)
	equal([]string{"TestRunSuite/TestSuite/TestFoo/subtest"}, pluginAfterEachSub)

	equal([]string{
		"TestRunSuite/TestSuite/TestBar",
		"TestRunSuite/TestSuite/TestFoo",
	}, afterEach)
	equal([]string{
		"TestRunSuite/TestSuite/TestBar",
		"TestRunSuite/TestSuite/TestFoo",
	}, pluginAfterEach)

	equal([]string{"TestRunSuite/TestSuite"}, afterAll)
	equal([]string{"TestRunSuite/TestSuite"}, pluginAfterAll)
}

func (s *TestSuite) BeforeAll(t *TestT) {
	s.beforeAllTriggered = true

	beforeAll = append(beforeAll, t.Name())
}

func (s TestSuite) BeforeEach(t *TestT) {
	if !s.beforeAllTriggered {
		t.Error("before all not triggered")
	}

	beforeEach = append(beforeEach, t.Name())
}

func (s TestSuite) AfterEach(t *TestT) {
	if !s.beforeAllTriggered {
		t.Error("before all not triggered")
	}

	afterEach = append(afterEach, t.Name())
}

func (s TestSuite) AfterAll(t *TestT) {
	if !s.beforeAllTriggered {
		t.Error("before all not triggered")
	}

	afterAll = append(afterAll, t.Name())
}

func (s TestSuite) TestFoo(t *TestT) {
	Run(t, "subtest", func(t *TestT) {})
}

func (s *TestSuite) TestBar(t *TestT) {}
