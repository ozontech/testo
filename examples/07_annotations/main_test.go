//go:build example

package main

import (
	"testing"

	"github.com/ozontech/testo"
)

type T struct {
	*testo.T

	// This is an example plugin that supports annotations
	*PluginUtils
}

type Suite struct{ testo.Suite[T] }

// here we instruct that TestFoo should be executed last, three times
var _ = testo.For(Suite.TestFoo, WithOrder(Last), WithRuns(3))

func (Suite) TestFoo(t T) {}

func (Suite) TestBar(t T) {}

var _ = testo.For(Suite.TestFizz, WithOrder(First))

func (Suite) TestFizz(t T) {}

func Test(t *testing.T) {
	testo.RunSuite(t, new(Suite))
}
