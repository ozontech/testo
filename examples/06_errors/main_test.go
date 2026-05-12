//go:build example

package main

import (
	"testing"

	"github.com/ozontech/testo"
)

type T struct{ *testo.T }

type MissingCases struct{ testo.Suite[T] }

// testo: wrong param signature for (*main.SuiteMissingCases).Test: missing (*main.SuiteMissingCases).CasesFoo() []int for param "Foo"
func (MissingCases) Test(t T, p struct{ Foo int }) {}

type InvalidCases struct{ testo.Suite[T] }

func (InvalidCases) CasesFoo() []string { return []string{"one", "two"} }

// testo: wrong param signature for (*main.InvalidCases).Test: (*main.InvalidCases).CasesFoo provides string values, not assignable to param "Foo" of type int
func (InvalidCases) Test(t T, p struct{ Foo int }) {}

type EmptyCases struct{ testo.Suite[T] }

func (EmptyCases) CasesFoo() []int { return nil }

// testo: warning: (*main.EmptyCases).CasesFoo provides zero values, (*main.EmptyCases).Test will not run
func (EmptyCases) Test(t T, p struct{ Foo int }) {}

type WrongT struct{ testo.Suite[T] }

// testo: wrong signature for (*main.WrongT).Test, must be: func (*main.WrongT).Test(main.T) or func (*main.WrongT).Test(main.T, struct{...})
func (WrongT) Test(t *testing.T) {}

func Test(t *testing.T) {
	t.Run("missing cases", func(t *testing.T) { testo.RunSuite(t, new(MissingCases)) })
	t.Run("invalid cases", func(t *testing.T) { testo.RunSuite(t, new(InvalidCases)) })
	t.Run("empty cases", func(t *testing.T) { testo.RunSuite(t, new(EmptyCases)) })
	t.Run("wrong t", func(t *testing.T) { testo.RunSuite(t, new(WrongT)) })
}
