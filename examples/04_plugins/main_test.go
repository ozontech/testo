//go:build example

package main

import (
	"testing"

	"github.com/ozontech/testo"
)

func Test(t *testing.T) {
	testo.RunSuite(t, new(Suite))
}

type T struct {
	*testo.T

	// install plugins by embedding them

	*ReverseTestsOrder
	*OverrideLog
	*AddNewMethods
	*Timer
	*RunsInnerSubtest
}

type Suite struct{ testo.Suite[T] }

func (Suite) TestFoo(t T) {
	testo.Run(t, "subtest", func(t T) {
		t.Log("Hello!")
	})
}

func (Suite) CasesN() []int {
	return []int{1, 2, 3}
}

func (Suite) TestBar(t T, params struct{ N int }) {
	t.Logf("params: %v", params)

	t.RunInnerSubtest()
}
