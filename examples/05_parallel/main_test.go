//go:build example

package main

import (
	"fmt"
	"testing"

	"github.com/ozontech/testo"
)

type T = *testo.T

type Suite struct{ testo.Suite[T] }

func Test(t *testing.T) {
	testo.RunSuite(t, new(Suite))
}

func (Suite) BeforeAll(t T) {
	t.Log("starting suite")
}

func (Suite) AfterAll(t T) {
	t.Log("finishing suite")
}

func (Suite) BeforeEach(t T) {
	t.Cleanup(func() {
		t.Logf("cleanup %q", t.Name())
	})

	t.Logf("before %q", t.Name())
}

func (Suite) AfterEach(t T) {
	t.Logf("after %q", t.Name())
}

func (Suite) TestFoo(t T) {
	// this will successfully convert this test into parallel one
	t.Parallel()

	t.Log("inside test foo")

	testo.Run(t, "sub-test", func(t T) {
		t.Parallel()

		for i := range 10 {
			testo.Run(t, fmt.Sprintf("inner %d", i), func(t T) {
				t.Parallel()

				t.Logf("inside %q", t.Name())
			})
		}
	})
}

func (Suite) TestBar(t T) {
	t.Parallel()

	t.Log("inside test bar")
}
