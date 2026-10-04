package testoplugin

// Hook is the plugin hook.
type Hook struct {
	// Priority defines execution order.
	// Lower values indicate that this hook should be run earlier than others and vice versa.
	// Zero value won't affect the order - it uses stable sort internally.
	//
	// See [TryFirst] and [TryLast] for predefined priority constants.
	Priority Priority

	// Func to be run for this hook.
	Func func()
}

// Hooks defines all hooks a plugin can define.
type Hooks struct {
	// BeforeAll is called before all tests once.
	BeforeAll Hook

	// BeforeEach is called before each test.
	BeforeEach Hook

	// BeforeEachSub is called before each subtest.
	BeforeEachSub Hook

	// AfterEachSub is called after each subtest.
	//
	// WARN: this hook is deferred to run at the end of the test.
	// If that test has sub-tests marked as parallel,
	// this hook will run BEFORE those sub-tests are finished.
	//
	// Unless you need to run sub-tests during this hook,
	// it is recommended to use t.Cleanup during BeforeEachSub.
	AfterEachSub Hook

	// AfterEach is called after each test.
	//
	// NOTE: this hook is deferred to run at the end of the test.
	// If that test has sub-tests marked as parallel,
	// this hook will run BEFORE those sub-tests are finished.
	//
	// Unless you need to run sub-tests during this hook,
	// it is recommended to use t.Cleanup during BeforeEach.
	AfterEach Hook

	// AfterAll is called after all tests once.
	AfterAll Hook
}
