package testo

import (
	"maps"
	"slices"
	"testing"
)

func Test_casesPermutations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input map[string][]int
		want  []map[string]int
	}{
		{
			name: "multiple keys",
			input: map[string][]int{
				"A": {1, 2},
				"B": {3},
			},
			want: []map[string]int{
				{"A": 1, "B": 3},
				{"A": 2, "B": 3},
			},
		},
		{
			name:  "empty input",
			input: map[string][]int{},
			want: []map[string]int{
				{},
			},
		},
		{
			name: "only one empty key",
			input: map[string][]int{
				"A": {1, 2, 3},
				"B": nil,
			},
			want: []map[string]int{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := casesPermutations(tc.input)

			ok := slices.EqualFunc(
				tc.want,
				got,
				func(a, b map[string]int) bool { return maps.Equal(a, b) },
			)

			if !ok {
				t.Errorf("not equal: want %+v, got %+v", tc.want, got)
			}
		})
	}
}
