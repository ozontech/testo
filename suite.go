package testo

type singleton[T CommonT] struct {
	Suite[T]

	name string
	test func(t T)
}

type suite[T CommonT] interface {
	// BeforeAll is called before all suite tests once.
	// T is shared with a top-level suite test.
	// Failing here will fail the entire suite.
	BeforeAll(t T)

	// BeforeEach is called before each suite test.
	// T is shared with an actual test.
	BeforeEach(t T)

	// AfterEach is called after each suite test.
	// T is shared with an actual test.
	//
	// WARN: this hook is defered to run at the end of the test.
	// If that test has sub-tests marked as parallel,
	// this hook will run BEFORE those sub-tests are finished.
	//
	// Unless you need to run sub-tests during this hook,
	// it is recommended to use t.Cleanup during BeforeEach.
	AfterEach(t T)

	// AfterAll is called after all suite tests once.
	// T is shared with a top-level suite test.
	// Failing here will fail the entire suite.
	AfterAll(t T)

	// A private method to prevent users implementing the
	// interface and so future additions to it will not
	// violate backwards compatibility.
	private()
}

// Suite is the base suite that all user-defined
// suites must embed.
//
// Suite may optionally provide hooks by implementing their methods:
//
//   - BeforeAll(T) - is called before all suite tests once.
//   - BeforeEach(T) - is called before each suite test. T is shared with an actual test.
//   - AfterEach(T) - is called after each suite test. T is shared with an actual test.
//   - AfterAll(T) - is called after all suite tests once.
//
// Example:
//
//	type MySuite struct {
//		testo.Suite[MyT]
//	}
type Suite[T CommonT] struct {
	// Mention *T in a field to disallow conversion between Pointer types.
	// See go.dev/issue/56603 for more details.
	// Use *T, not T, to avoid spurious recursive type definition errors.
	_ [0]*T
}

// BeforeAll hook.
func (Suite[T]) BeforeAll(t T) {
	t.unwrap().reflection.Suite.Hooks.MissedBeforeAll = true
}

// BeforeEach hook.
func (Suite[T]) BeforeEach(t T) {
	t.unwrap().reflection.Suite.Hooks.MissedBeforeEach = true
}

// AfterEach hook.
func (Suite[T]) AfterEach(t T) {
	t.unwrap().reflection.Suite.Hooks.MissedAfterEach = true
}

// AfterAll hook.
func (Suite[T]) AfterAll(t T) {
	t.unwrap().reflection.Suite.Hooks.MissedAfterAll = true
}

func (Suite[T]) private() {}
