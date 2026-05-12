package stack

import (
	"slices"
)

// Stack is a first-in-last-out (FILO) data structure.
type Stack[T any] struct {
	values []T
}

func New[T any]() Stack[T] {
	return Stack[T]{}
}

func (s *Stack[T]) Len() int {
	return len(s.values)
}

func (s *Stack[T]) Push(v T) {
	s.values = append(s.values, v)
}

func (s *Stack[T]) Pop() (T, bool) {
	if len(s.values) == 0 {
		return *new(T), false
	}

	last := s.values[len(s.values)-1]

	s.values = slices.Delete(s.values, len(s.values)-1, len(s.values))

	return last, true
}
