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
	parent     *MockPluginWithoutT
	options    []testoplugin.Option
	initCalled int
}

func (m *MockPluginWithoutT) Plugin(
	parent testoplugin.Plugin,
	options ...testoplugin.Option,
) testoplugin.Spec {
	m.parent = parent.(*MockPluginWithoutT)
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

func mustPanic(t *testing.T, f func()) {
	t.Helper()

	defer func() { recover() }()

	f()

	t.Fatal("function did not panic")
}
