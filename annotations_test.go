package testo

import (
	"reflect"
	"testing"

	"github.com/ozontech/testo/testoplugin"
)

func withTestValue(s string) testoplugin.Option {
	return testoplugin.Option{
		Value: s,
	}
}

type AnnotatedSuite struct{ Suite[*T] }

var _ = For(AnnotatedSuite.TestFoo, withTestValue("test-foo"))

func (AnnotatedSuite) TestFoo(*T) {}

var _ = For(AnnotatedSuite.TestBar, withTestValue("test-bar"))

func (AnnotatedSuite) TestBar(*T) {}

var _ = For((*AnnotatedSuite).TestFizz, withTestValue("test-fizz"))

func (*AnnotatedSuite) TestFizz(*T) {}

var _ = ForEach(AnnotatedSuite.TestBuzz, withTestValue("test-buzz"))

func (AnnotatedSuite) TestBuzz(*T, struct{}) {}

func TestFor(t *testing.T) {
	wantTestValue := func(want string, v reflect.Value) {
		t.Helper()

		id := getID[AnnotatedSuite](v)

		annotations := annotationsFor(id)

		if len(annotations) != 1 {
			t.Fatalf("want exactly 1 annotation, got %d", len(annotations))
		}

		got, ok := annotations[0].Value.(string)
		if !ok {
			t.Fatalf("want string value, got %T", annotations[0].Value)
		}

		if want != got {
			t.Fatalf("want %q, got %q", want, got)
		}
	}

	wantTestValue("test-foo", reflect.ValueOf(AnnotatedSuite.TestFoo))
	wantTestValue("test-bar", reflect.ValueOf(AnnotatedSuite.TestBar))
	wantTestValue("test-fizz", reflect.ValueOf((*AnnotatedSuite).TestFizz))
	wantTestValue("test-buzz", reflect.ValueOf(AnnotatedSuite.TestBuzz))
}
