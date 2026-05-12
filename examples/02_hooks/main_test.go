//go:build example

package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/ozontech/testo"
)

func Test(t *testing.T) {
	testo.RunSuite(t, new(Suite))
}

type T = *testo.T

type Suite struct {
	testo.Suite[T]

	client *http.Client
}

// ======== Hooks ==========

func (s *Suite) BeforeAll(t T) {
	s.client = &http.Client{Timeout: 20 * time.Second}
}

func (*Suite) BeforeEach(t T) {
	t.Logf("Marking test %q as parallel", t.Name())
	t.Parallel()
}

func (*Suite) AfterEach(t T) {
	if t.Failed() {
		t.Logf("Test %q failed", t.Name())
	}
}

func (s *Suite) AfterAll(t T) {
	s.client.CloseIdleConnections()
}

// ======== Actual tests ==========

func (s *Suite) TestFoo(t T) {
	res, err := s.client.Get("https://example.org")
	if err != nil {
		t.Fatal("failed to example website", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Error("status code is not ok", err)
	}
}

func (s *Suite) TestBar(t T) {
	t.Log("Hello!")
}
