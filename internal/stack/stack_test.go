package stack

import (
	"reflect"
	"testing"
)

func TestStack_PushPop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pushes   []int
		pops     int
		wantVals []int
		wantOks  []bool
	}{
		{
			name:     "single element",
			pushes:   []int{42},
			pops:     2,
			wantVals: []int{42, 0},
			wantOks:  []bool{true, false},
		},
		{
			name:     "multiple elements",
			pushes:   []int{1, 2, 3},
			pops:     4,
			wantVals: []int{3, 2, 1, 0},
			wantOks:  []bool{true, true, true, false},
		},
		{
			name:     "boundary zeros",
			pushes:   []int{0, 0},
			pops:     3,
			wantVals: []int{0, 0, 0},
			wantOks:  []bool{true, true, false},
		},
		{
			name:     "no pushes",
			pushes:   []int{},
			pops:     1,
			wantVals: []int{0},
			wantOks:  []bool{false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var s Stack[int]

			for _, v := range tt.pushes {
				s.Push(v)
			}

			gotVals := make([]int, 0, tt.pops)
			gotOks := make([]bool, 0, tt.pops)

			for range tt.pops {
				v, ok := s.Pop()
				gotVals = append(gotVals, v)
				gotOks = append(gotOks, ok)
			}

			if !reflect.DeepEqual(tt.wantVals, gotVals) {
				t.Error("values popped sequence mismatch")
			}

			if !reflect.DeepEqual(tt.wantOks, gotOks) {
				t.Error("ok flags sequence mismatch")
			}
		})
	}
}
