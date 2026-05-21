//go:build example

package main

import (
	"testing"

	"github.com/ozontech/testo"
)

type T = *testo.T

func TestSimple(t *testing.T) {
	testo.RunTest(t, func(t T) {
		t.Log("Hello from testo!")
	})
}

func TestMultiple(t *testing.T) {
	t.Run("First test", testo.Test(func(t T) {
		t.Log("Hello from the first test!")
	}))

	t.Run("Second test", testo.Test(func(t T) {
		t.Log("Hello from the second test!")
	}))
}
