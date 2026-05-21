package testo

import (
	"reflect"
	"slices"
	"testing"
)

type MySuite struct{ Suite[*T] }

func (MySuite) CasesNumbers() []int {
	return []int{1, 2, 3}
}

func (MySuite) CasesEmpty() []string {
	return []string{}
}

func TestSuiteCasesOf(t *testing.T) {
	t.Parallel()

	cases := suiteCasesOf[MySuite](t)

	// Expect two case sets: "Numbers" and "Empty"
	if _, ok := cases["Numbers"]; !ok {
		t.Error("case set for Numbers not found")
	}

	if _, ok := cases["Empty"]; !ok {
		t.Error("case set for Empty not found")
	}

	// Test "Numbers" provides type int
	numCase := cases["Numbers"]
	if reflect.TypeFor[int]() != numCase.Provides {
		t.Error("Numbers case does not provide int")
	}

	// Run the case function
	numVals := numCase.Func(MySuite{})
	ints := make([]int, len(numVals))

	for i, rv := range numVals {
		ints[i] = int(rv.Int())
	}

	if !slices.Equal(ints, []int{1, 2, 3}) {
		t.Error("case values for Numbers does not match")
	}

	// Test "Empty" provides type string
	emptyCase := cases["Empty"]

	if reflect.TypeFor[string]() != emptyCase.Provides {
		t.Error("Empty case does not provide string")
	}

	// Run the case function
	emptyVals := emptyCase.Func(MySuite{})

	if len(emptyVals) > 0 {
		t.Error("Empty case is not empty")
	}
}

func BenchmarkCasesPermutations(b *testing.B) {
	// Small input: 2 keys with 2 values each (4 permutations)
	b.Run("small", func(b *testing.B) {
		input := map[string][]int{
			"a": {1, 2},
			"b": {3, 4},
		}
		b.ResetTimer()
		for range b.N {
			casesPermutations(input)
		}
	})

	// Medium input: 3 keys with 3 values each (27 permutations)
	b.Run("medium", func(b *testing.B) {
		input := map[string][]int{
			"a": {1, 2, 3},
			"b": {4, 5, 6},
			"c": {7, 8, 9},
		}
		b.ResetTimer()
		for range b.N {
			casesPermutations(input)
		}
	})

	// Large input: 5 keys with 5 values each (3_125 permutations)
	b.Run("large", func(b *testing.B) {
		input := map[string][]int{
			"a": {1, 2, 3, 4, 5},
			"b": {6, 7, 8, 9, 10},
			"c": {11, 12, 13, 14, 15},
			"d": {16, 17, 18, 19, 20},
			"e": {21, 22, 23, 24, 25},
		}
		b.ResetTimer()
		for range b.N {
			casesPermutations(input)
		}
	})
}

type SubSuiteParent struct {
	Suite[*T]
}

func (s SubSuiteParent) Test(t *T) {
	if !RunSubSuite(t, new(SubSuiteChild)) {
		t.Fatal("run sub suite failed")
	}
}

type SubSuiteChild struct{ Suite[*T] }

func (s SubSuiteChild) Test(t *T) {
	if reflect.TypeOf(Reflect(t).Suite.Parent.Value) != reflect.TypeFor[*SubSuiteParent]() {
		t.Fatal("unexpected parent suite type")
	}
}

func TestSubSuite(t *testing.T) {
	t.Parallel()

	if !RunSuite(t, new(SubSuiteParent)) {
		t.Fatal("run suite failed")
	}
}
