package testo

import (
	"reflect"
	"slices"
	"testing"
	"unsafe"

	"github.com/ozontech/testo/testoplugin"
)

type MockT struct {
	*T

	*MockPluginWithT
	*MockPluginWithoutT
	Other *MockPluginWithT
}

type MockPluginWithT struct {
	*T
	initCalled int
}

func (m *MockPluginWithT) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
	m.initCalled++

	return testoplugin.Spec{}
}

type MockPluginWithoutT struct {
	parent       *MockPluginWithoutT
	parentNonNil bool
	options      []testoplugin.Option
	initCalled   int
}

func (m *MockPluginWithoutT) Plugin(
	parent testoplugin.Plugin,
	options ...testoplugin.Option,
) testoplugin.Spec {
	// For top-level tests parent is a typed-nil *MockPluginWithoutT inside a
	// non-nil interface, as documented in [testoplugin.Plugin]. Recording the
	// interface comparison separately from the asserted pointer lets
	// TestConstruct distinguish typed-nil from a true nil interface.
	m.parentNonNil = parent != nil
	if parent != nil {
		m.parent = parent.(*MockPluginWithoutT)
	}

	m.options = options
	m.initCalled++

	return testoplugin.Spec{}
}

type MockPluginWithNonPointerT struct{ T }

type InvalidNonPointerTT struct {
	*T

	*MockPluginWithNonPointerT
}

type InvalidRecursiveT struct {
	*T

	*InvalidRecursiveT
}

type InvalidNonPointerPluginT struct {
	*T

	MockPluginWithT
}

func TestConstruct(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		options := []testoplugin.Option{
			{Value: "foo", Propagate: true},
			{Value: "bar", Propagate: false},
		}

		t.Log("constructing root")

		res := construct[MockT](t, nil, nil, options...)

		t.Log("constructing child")

		child := construct(t, &res, nil, testoplugin.Option{Value: "fizz"})

		if !slices.Equal([]testoplugin.Option{
			{Value: "foo", Propagate: true},
			{Value: "bar", Propagate: false},
		}, res.levelOptions) {
			t.Error("level options not equal")
		}

		if !reflect.DeepEqual(res.T, res.MockPluginWithT.T) {
			t.Error("res.T and res.MockPluginWithT.T not equal")
		}

		if res.MockPluginWithT.initCalled != 1 {
			t.Error("res.MockPluginWithT.initCalled not equal to 1")
		}

		if res.MockPluginWithoutT.initCalled != 1 {
			t.Error("res.MockPluginWithoutT.initCalled not equal to 1")
		}

		if res.Other == nil {
			t.Fatal("res.Other is nil")
		}

		if !reflect.DeepEqual(res.T, res.Other.T) {
			t.Error("res.T not equal res.Other.T")
		}

		for i, counter := range []int{
			res.Other.initCalled,
			child.MockPluginWithoutT.initCalled,
			child.MockPluginWithT.initCalled,
			child.Other.initCalled,
		} {
			if counter != 1 {
				t.Errorf("counter #%d: initCalled not equal to 1", i)
			}
		}

		// Top level: parent must be a typed-nil instance of the plugin type -
		// a non-nil interface wrapping a nil pointer - so that plugins doing
		// an unconditional parent.(*MyPlugin) assertion keep working.
		if !res.MockPluginWithoutT.parentNonNil {
			t.Error(
				"res.MockPluginWithoutT: top-level parent interface is nil, want typed-nil instance",
			)
		}

		if res.MockPluginWithoutT.parent != nil {
			t.Error("res.MockPluginWithoutT.parent is not a nil pointer for a top-level test")
		}

		if child.MockPluginWithoutT.parent != res.MockPluginWithoutT {
			t.Error("child.MockPluginWithoutT.parent does not point to the parent's plugin")
		}

		if !reflect.DeepEqual(res.T, child.T.parent) {
			t.Error("res.T not equal to child.T.parent")
		}

		if reflect.DeepEqual(res, child) {
			t.Error("res is equal to child")
		}

		if !slices.Equal([]testoplugin.Option{
			{Value: "foo", Propagate: true},
			{Value: "fizz"},
		}, child.MockPluginWithoutT.options) {
			t.Error("child.MockPluginWithoutT.options not equal to expected")
		}

		if unsafe.Pointer(child.Other) != unsafe.Pointer(child.MockPluginWithT) {
			t.Error("child.Other does not point to child.MockPluginWithT")
		}
	})

	t.Run("non pointer t", func(t *testing.T) {
		t.Parallel()

		mustPanic(t, func() {
			construct[InvalidNonPointerTT](t, nil, nil)
		})
	})

	t.Run("recursive t", func(t *testing.T) {
		t.Parallel()

		mustPanic(t, func() {
			construct[InvalidRecursiveT](t, nil, nil)
		})
	})

	t.Run("non pointer plugin", func(t *testing.T) {
		t.Parallel()

		mustPanic(t, func() {
			construct[InvalidNonPointerPluginT](t, nil, nil)
		})
	})
}

// nestedPluginCalls records the order in which Plugin is called for the mock
// plugins below. Only TestConstructNestedPluginOrder and its plugins touch it.
var nestedPluginCalls []string

type MockInnerPlugin struct{ initCalled int }

func (m *MockInnerPlugin) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
	m.initCalled++
	nestedPluginCalls = append(nestedPluginCalls, "Inner")

	return testoplugin.Spec{}
}

type MockOuterPlugin struct {
	Inner *MockInnerPlugin

	initCalled int
}

func (m *MockOuterPlugin) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
	m.initCalled++
	nestedPluginCalls = append(nestedPluginCalls, "Outer")

	return testoplugin.Spec{}
}

type MockDeepPlugin struct{ initCalled int }

func (m *MockDeepPlugin) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
	m.initCalled++
	nestedPluginCalls = append(nestedPluginCalls, "Deep")

	return testoplugin.Spec{}
}

type MockMidPlugin struct {
	Deep *MockDeepPlugin

	initCalled int
}

func (m *MockMidPlugin) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
	m.initCalled++
	nestedPluginCalls = append(nestedPluginCalls, "Mid")

	return testoplugin.Spec{}
}

type NestedMockT struct {
	*T

	Outer *MockOuterPlugin
	Mid   *MockMidPlugin
}

func TestConstructNestedPluginOrder(t *testing.T) {
	t.Parallel()

	nestedPluginCalls = nil

	res := construct[NestedMockT](t, nil, nil)

	if res.Outer.Inner == nil {
		t.Fatal("res.Outer.Inner is nil")
	}

	if res.Mid.Deep == nil {
		t.Fatal("res.Mid.Deep is nil")
	}

	for _, counter := range []struct {
		name  string
		value int
	}{
		{"Outer", res.Outer.initCalled},
		{"Inner", res.Outer.Inner.initCalled},
		{"Mid", res.Mid.initCalled},
		{"Deep", res.Mid.Deep.initCalled},
	} {
		if counter.value != 1 {
			t.Errorf("%s: initCalled = %d, want 1", counter.name, counter.value)
		}
	}

	// Plugin must be called on a nested plugin before its container, so that
	// the container observes fully initialized nested plugins. Sibling
	// subtrees have no guaranteed relative order, so only pairwise
	// nested-before-container positions are asserted.
	for _, pair := range []struct{ nested, container string }{
		{"Inner", "Outer"},
		{"Deep", "Mid"},
	} {
		nestedIdx := slices.Index(nestedPluginCalls, pair.nested)
		containerIdx := slices.Index(nestedPluginCalls, pair.container)

		if nestedIdx < 0 || containerIdx < 0 {
			t.Fatalf("missing Plugin calls in %v", nestedPluginCalls)
		}

		if nestedIdx > containerIdx {
			t.Errorf(
				"Plugin called for container %s before nested %s: %v",
				pair.container, pair.nested, nestedPluginCalls,
			)
		}
	}

	// pluginOrder drives spec merging and must stay in declaration order:
	// containers before their nested plugins, siblings as declared in T.
	wantOrder := []reflect.Type{
		reflect.TypeFor[*T](),
		reflect.TypeFor[*MockOuterPlugin](),
		reflect.TypeFor[*MockInnerPlugin](),
		reflect.TypeFor[*MockMidPlugin](),
		reflect.TypeFor[*MockDeepPlugin](),
	}

	if !slices.Equal(wantOrder, res.unwrap().pluginOrder) {
		t.Errorf("pluginOrder = %v, want %v", res.unwrap().pluginOrder, wantOrder)
	}
}

func mustPanic(t *testing.T, f func()) {
	t.Helper()

	defer func() { recover() }()

	f()

	t.Fatal("function did not panic")
}
