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

type MalformedCases struct{ testo.Suite[T] }

// testo: (*main.MalformedCases).Casesfoo has malformed name: first letter after 'Cases' must not be lowercase
func (MalformedCases) Casesfoo() []string { return []string{"a", "b"} }

func (MalformedCases) Test(t T, p struct{ Foo string }) {}

type MalformedTest struct{ testo.Suite[T] }

// testo: (*main.MalformedTest).Testimony has malformed name: first letter after 'Test' must not be lowercase
func (MalformedTest) Testimony(t T) {}

type EmptyCases struct{ testo.Suite[T] }

func (EmptyCases) CasesFoo() []int { return nil }

// testo: warning: (*main.EmptyCases).CasesFoo provides zero values, (*main.EmptyCases).Test won't run
func (EmptyCases) Test(t T, p struct{ Foo int }) {}

type WrongT struct{ testo.Suite[T] }

// testo: wrong signature for (*main.WrongT).Test, must be: func (*main.WrongT).Test(main.T) or func (*main.WrongT).Test(main.T, struct{...})
func (WrongT) Test(t *testing.T) {}

// testo: warning: suite *main.MissingTests has no tests
type MissingTests struct{ testo.Suite[T] }

func Test(t *testing.T) {
	t.Run("malformed cases", func(t *testing.T) { testo.RunSuite(t, new(MalformedCases)) })
	t.Run("malformed test", func(t *testing.T) { testo.RunSuite(t, new(MalformedTest)) })
	t.Run("missing cases", func(t *testing.T) { testo.RunSuite(t, new(MissingCases)) })
	t.Run("invalid cases", func(t *testing.T) { testo.RunSuite(t, new(InvalidCases)) })
	t.Run("empty cases", func(t *testing.T) { testo.RunSuite(t, new(EmptyCases)) })
	t.Run("wrong t", func(t *testing.T) { testo.RunSuite(t, new(WrongT)) })
	t.Run("missing tests", func(t *testing.T) { testo.RunSuite(t, new(MissingTests)) })
}
