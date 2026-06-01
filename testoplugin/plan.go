package testoplugin

import (
	"github.com/ozontech/testo/internal/pragma"
	"github.com/ozontech/testo/testoreflect"
)

// Plan for running the tests.
type Plan struct {
	// Priority sets plan priority.
	// Plans with lower priority value are executed first.
	Priority Priority

	// Prepare may filter or re-order planned tests in-place.
	// Nil values are ignored.
	//
	// This function will be called once before running any top-level tests.
	// It will not receive subtests.
	Prepare func(suite testoreflect.SuiteInfo, tests *[]PlannedTest)
}

// PlannedTest is a test to be scheduled for execution.
type PlannedTest interface {
	pragma.DoNotImplement

	// Info about this test.
	Info() testoreflect.TestInfo

	// Annotations of this test.
	Annotations() []Option
}
