package reflectutil

import (
	"reflect"
	"testing"
)

func TestFindRecursiveType(t *testing.T) {
	t.Parallel()

	t.Run("without recursive", func(t *testing.T) {
		t.Parallel()

		type Bar struct {
			Field string

			private int
		}

		type Foo struct {
			Field string

			Bar
		}

		got := FindRecursiveType(reflect.TypeFor[Foo]())

		if got != nil {
			t.Fatalf("want nil, got %v", got)
		}
	})

	t.Run("with recursive", func(t *testing.T) {
		t.Parallel()

		type Bar struct {
			Field string

			*Bar
		}

		type Foo struct {
			private int

			Bar

			Field string
		}

		found := FindRecursiveType(reflect.TypeFor[Foo]())

		if found != reflect.TypeFor[Bar]() {
			t.Fatalf("want Bar, got %s", found)
		}
	})
}

func TestName(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		Want string
		Got  string
	}{
		{
			Want: "T",
			Got:  NameOf[******testing.T](),
		},
		{
			Want: "T",
			Got:  NameOf[testing.T](),
		},
		{
			Want: "string",
			Got:  NameOf[string](),
		},
	} {
		if tt.Want != tt.Got {
			t.Errorf("want %q, got %q", tt.Want, tt.Got)
		}
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	type Mock struct {
		N   int
		Ptr *int
	}

	t.Run("non-pointer", func(t *testing.T) {
		t.Parallel()

		mock := New[Mock]()

		if mock.N != 0 && mock.Ptr != nil {
			t.Error("invalid value returned")
		}
	})

	t.Run("pointer", func(t *testing.T) {
		t.Parallel()

		mock := New[*Mock]()

		if mock == nil {
			t.Fatal("new returned nil")
		}

		if mock.Ptr != nil {
			t.Error("new initialized ptr field")
		}
	})
}

func TestFilled(t *testing.T) {
	t.Parallel()

	t.Run("regular", func(t *testing.T) {
		type Mock struct {
			String *string
			Nested *struct {
				Nested *struct {
					N *int
				}
			}
		}

		value, ok := Filled(reflect.TypeFor[Mock]())

		if !ok {
			t.Fatal("ok must be true")
		}

		v := value.Interface().(Mock)

		if v.String == nil {
			t.Error("Mock.String is nil")
		}

		if v.Nested == nil {
			t.Fatal("Mock.Nested is nil")
		}

		if v.Nested.Nested == nil {
			t.Fatal("Mock.Nested.Nested is nil")
		}

		if v.Nested.Nested.N == nil {
			t.Fatal("Mock.Nested.Nested.N is nil")
		}
	})

	t.Run("self referencing pointer", func(t *testing.T) {
		type Node struct {
			Value int
			Next  *Node
		}

		value, ok := Filled(reflect.TypeFor[Node]())

		if ok {
			t.Error("ok should be false")
		}

		want := Node{
			Value: 0,
			Next:  new(Node),
		}
		got := value.Interface()

		if !reflect.DeepEqual(want, got) {
			t.Errorf("nodes not equal: want %+v, got %+v", want, got)
		}
	})

	t.Run("repeating type", func(t *testing.T) {
		type Bar struct {
			X *int
		}

		type Foo struct {
			Value    int
			Bar      *Bar
			OtherBar *Bar
		}

		value, ok := Filled(reflect.TypeFor[Foo]())

		if !ok {
			t.Error("ok should be true")
		}

		v := value.Interface().(Foo)

		if v.Bar == nil {
			t.Fatal("Foo.Bar is nil")
		}

		if v.Bar.X == nil {
			t.Fatal("Foo.Bar.X is nil")
		}

		if v.OtherBar == nil {
			t.Fatal("Foo.OtherBar is nil")
		}

		if v.OtherBar.X == nil {
			t.Fatal("Foo.OtherBar.X is nil")
		}
	})
}
