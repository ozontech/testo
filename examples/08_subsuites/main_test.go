//go:build example

package main

import (
	"testing"

	"github.com/ozontech/testo"
)

type T = *testo.T

type OuterSuite struct{ testo.Suite[T] }

func (OuterSuite) Test(t T) {
	testo.RunSubSuite(t, new(InnerSuite))
}

type InnerSuite struct{ testo.Suite[T] }

func (InnerSuite) Test(t T) {
	t.Log("Hello from sub-suite!")
}

func Test(t *testing.T) {
	testo.RunSuite(t, new(OuterSuite))
}
