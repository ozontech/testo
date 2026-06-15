//go:build example

package main

import (
	"testing"

	"github.com/ozontech/testo"
)

type T struct{ *testo.T }

// Test connects `go test` with our testo suite.
func Test(t *testing.T) {
	testo.RunSuite(t, new(Suite))
}

type Suite struct {
	testo.Suite[T]
}

func (*Suite) CasesAboba() []uint {
	return nil
}

func (s *Suite) TestiMath(t T) {
	if 2+2 != 4 {
		t.Errorf("expected 2 + 2 to be 4, got: %d", 2+2)
	}
}
