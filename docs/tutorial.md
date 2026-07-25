# Tutorial

In this tutorial we take a plain Go test and grow it, step by step,
into a full Testo suite with a plugin, parametrization, hooks,
fixtures and steps.

Each section changes only a small piece of code, and the file compiles
and passes after every change. The complete final file is listed
[at the end](#the-finished-test-file).

## Setting Up

Create a new project and add Testo to it:

```bash
mkdir testo-bakery
cd testo-bakery
go mod init testo-bakery
go get github.com/ozontech/testo
```

We are going to test a tiny bakery. Create `main.go`:

```go
package main

// Pastry is something a bakery bakes.
type Pastry struct {
	Name        string
	Tasty       bool
	Ingredients string
}

var recipes = map[string]string{
	"honey cake": "honey, flour, eggs, sour cream",
	"tiramisu":   "mascarpone, coffee, ladyfingers",
}

// Bake bakes a pastry, if the bakery knows the recipe.
func Bake(name string) (Pastry, bool) {
	ingredients, ok := recipes[name]
	if !ok {
		return Pastry{}, false
	}

	return Pastry{Name: name, Tasty: true, Ingredients: ingredients}, true
}

// Eat disposes of a pastry in the most pleasant way possible.
func Eat(Pastry) {}

// Oven turns the bakery oven on or off.
func Oven(on bool) error { return nil }

func main() {}
```

## A Plain Go Test

We start without Testo. Create `main_test.go`:

```go
package main

import "testing"

func TestBake(t *testing.T) {
	t.Run("HoneyCake", func(t *testing.T) {
		pastry, ok := Bake("honey cake")
		if !ok {
			t.Fatal("the bakery must know how to bake a honey cake")
		}

		// no reason to waste it after the test
		t.Cleanup(func() { Eat(pastry) })

		if !pastry.Tasty {
			t.Error("honey cake must be tasty")
		}
	})
}
```

```bash
go test . -v
```

```txt
=== RUN   TestBake
=== RUN   TestBake/HoneyCake
--- PASS: TestBake (0.00s)
    --- PASS: TestBake/HoneyCake (0.00s)
PASS
```

So far this is a plain Go test. Let's bring in Testo.

## Switching to Testo

Replace the `t.Run` call with `testo.RunTest`:

```go
package main

import (
	"testing"

	"github.com/ozontech/testo"
)

func TestBake(t *testing.T) {
	testo.RunTest(t, func(t *testo.T) {
		pastry, ok := Bake("honey cake")
		if !ok {
			t.Fatal("the bakery must know how to bake a honey cake")
		}

		// no reason to waste it after the test
		t.Cleanup(func() { Eat(pastry) })

		if !pastry.Tasty {
			t.Error("honey cake must be tasty")
		}
	})
}
```

The body of the test did not change - only the type of `t`.
The `HoneyCake` name is gone, though: `RunTest` numbers its tests
instead. Suites (below) give tests proper names again.

`testo.T` wraps `testing.T` and keeps its interface, with one
exception: sub-tests are started with `testo.Run` instead of `t.Run`
(covered [later](#steps-sub-tests)). It also implements `testing.TB`,
so assertion libraries and mocks built for standard tests keep
working.

Run the test again:

```txt
=== RUN   TestBake
=== RUN   TestBake/#00
=== RUN   TestBake/#00/testo!
=== RUN   TestBake/#00/testo!/TestBake
--- PASS: TestBake (0.00s)
    --- PASS: TestBake/#00 (0.00s)
        --- PASS: TestBake/#00/testo! (0.00s)
            --- PASS: TestBake/#00/testo!/TestBake (0.00s)
PASS
```

Two technical levels appeared in the names:

- `#00` is the index of the test inside this `RunTest` call.
- `testo!` is a special test Testo inserts to guarantee correct work
  of hooks with parallel tests. It does not affect your tests, but it
  does affect `-run` patterns - see
  [how to run specific tests](./how-to.md#how-to-run-and-skip-specific-tests).

So far Testo has given us nothing new. Plugins are where it gets
interesting.

## Plugins

A plugin is a struct that describes what it does through the
`testoplugin.Plugin` interface:

```go
type Plugin interface {
	Plugin(parent Plugin, options ...Option) Spec
}

type Spec struct {
	// Plan tests for execution: filter, duplicate, reorder.
	Plan Plan

	// Hooks around suites, tests and sub-tests.
	Hooks Hooks

	// Middleware for built-in T methods,
	// such as Fail, Log, Skip and others.
	Overrides Overrides
}
```

Let's write a plugin that measures how long each test takes.
In `main_test.go`, update the import block as shown below and add
the plugin after it:

```go
import (
	"testing"
	"time"

	"github.com/ozontech/testo"
	"github.com/ozontech/testo/testoplugin"
)

type PluginTimer struct {
	*testo.T

	start time.Time
}

func (p *PluginTimer) Plugin(testoplugin.Plugin, ...testoplugin.Option) (spec testoplugin.Spec) {
	spec.Hooks.BeforeEach.Func = func() {
		p.start = time.Now()
	}

	spec.Hooks.AfterEach.Func = func() {
		p.Logf("test %q took %s", p.Name(), time.Since(p.start))
	}

	return spec
}
```

The plugin embeds `*testo.T`. Testo fills it with the same `T` as
the current test, so the plugin sees everything the test sees.
The unused first argument of the `Plugin` method is the parent plugin
instance - see the
[technical overview](./technical-overview.md#lifecycle) if you need it.

The `Plugin` method is called for each test and sub-test, and each
gets its own plugin instance. That is why writing to plugin fields
is safe.

To use the plugin, define your own `T` type that embeds `*testo.T`
together with the plugins you want:

```go
type T struct {
	*testo.T
	*PluginTimer
}
```

And change the test function to take this `T`:

```go
func TestBake(t *testing.T) {
	testo.RunTest(t, func(t T) {
		// ... body unchanged ...
	})
}
```

That is the whole installation. Testo looks at the `T` type a test
accepts, then collects and initializes the plugins listed in it:

```txt
=== RUN   TestBake
=== RUN   TestBake/#00
    main_test.go:35: testo: plugins collected: 1: main.PluginTimer
=== RUN   TestBake/#00/testo!
=== RUN   TestBake/#00/testo!/TestBake
    main_test.go:23: test "TestBake/#00/TestBake" took 98.125µs
--- PASS: TestBake (0.00s)
...
PASS
```

The logged name has no `testo!` in it: `t.Name()` returns the
logical test name
([details](./how-to.md#how-to-run-and-skip-specific-tests)).

> [!NOTE]
> Plugins must be embedded as pointers. Pointers let plugins share
> state with each other by pointing to the same memory.

Plugins can do much more than hooks: reorder or filter the test plan,
wrap built-in methods like `t.Log`, add new methods to `T`, accept
[options](./how-to.md#how-to-use-plugin-options) and command line
flags. The [plugins example](../examples/04_plugins/main_test.go)
shows `Plan` and `Overrides` in action, and the
[technical overview](./technical-overview.md#plugins) explains how
plugins talk to each other.

Ready-made plugins:
[testo-allure](https://github.com/ozontech/testo-allure) for Allure
reports, and [testo-toppings](https://github.com/ozontech/testo-toppings) -
a collection of small utility plugins.

For instance, here is a complete plugin that makes every test parallel:

```go
type PluginParallel struct{ *testo.T }

func (p *PluginParallel) Plugin(testoplugin.Plugin, ...testoplugin.Option) (spec testoplugin.Spec) {
	spec.Hooks.BeforeEach.Func = func() {
		p.Parallel()
	}

	return spec
}
```

Don't add this one to your file, or it will change the outputs
below. A more complete version is available as the
[parallel plugin](https://github.com/ozontech/testo-toppings/tree/main/parallel)
in testo-toppings.

## Suites

When tests share setup and plugins, it is convenient to group them
into a suite. A suite embeds `testo.Suite[T]`, where `T` is the type
we defined above - i.e. the set of plugins every test in the suite
gets:

```go
type Bakery struct{ testo.Suite[T] }
```

Tests become methods of the suite. They follow the usual Go naming
rules (the `Test` prefix) and must accept the same `T` as specified in
`testo.Suite[T]`. Value and pointer receivers both work.
Replace the `TestBake` function with:

```go
func (Bakery) TestBake(t T) {
	pastry, ok := Bake("honey cake")
	if !ok {
		t.Fatal("the bakery must know how to bake a honey cake")
	}

	t.Cleanup(func() { Eat(pastry) })

	if !pastry.Tasty {
		t.Error("honey cake must be tasty")
	}
}
```

Go has no native notion of suites, so we launch the suite from one
regular test function:

```go
func Test(t *testing.T) {
	testo.RunSuite(t, new(Bakery))
}
```

```txt
=== RUN   Test
=== RUN   Test/Bakery
    main_test.go:50: testo: plugins collected: 1: main.PluginTimer
=== RUN   Test/Bakery/testo!
=== RUN   Test/Bakery/testo!/TestBake
    main_test.go:23: test "Test/Bakery/TestBake" took 66.583µs
--- PASS: Test (0.00s)
    --- PASS: Test/Bakery (0.00s)
        --- PASS: Test/Bakery/testo! (0.00s)
            --- PASS: Test/Bakery/testo!/TestBake (0.00s)
PASS
```

## Parametrized Tests

Our bakery bakes more than honey cake, and the test scenario is the
same for every dessert. Instead of copying the test, we make the
dessert a parameter.

A parametrized test takes a second argument after `T`: a struct whose
fields are the parameters. Replace `TestBake` with:

```go
func (Bakery) TestBake(t T, p struct{ Dessert string }) {
	pastry, ok := Bake(p.Dessert)
	if !ok {
		t.Fatalf("the bakery must know how to bake %s", p.Dessert)
	}

	t.Cleanup(func() { Eat(pastry) })

	if !pastry.Tasty {
		t.Errorf("%s must be tasty", p.Dessert)
	}
}
```

Parameter values are declared with `CasesXxx` methods, where `Xxx`
matches the field name:

```go
func (Bakery) CasesDessert() []string {
	return []string{"honey cake", "tiramisu"}
}
```

```txt
=== RUN   Test
=== RUN   Test/Bakery
    main_test.go:54: testo: plugins collected: 1: main.PluginTimer
=== RUN   Test/Bakery/testo!
=== RUN   Test/Bakery/testo!/TestBake
    main_test.go:23: test "Test/Bakery/TestBake" took 108.791µs
=== RUN   Test/Bakery/testo!/TestBake#01
    main_test.go:23: test "Test/Bakery/TestBake#01" took 15.042µs
--- PASS: Test (0.00s)
    --- PASS: Test/Bakery (0.00s)
        --- PASS: Test/Bakery/testo! (0.00s)
            --- PASS: Test/Bakery/testo!/TestBake (0.00s)
            --- PASS: Test/Bakery/testo!/TestBake#01 (0.00s)
PASS
```

The test ran once per dessert. The name `TestBake#01` does not say
which dessert it was, so log `p.Dessert` at the top of the test if
that matters to you. With several parameters, the test runs
with the [Cartesian product](https://en.wikipedia.org/wiki/Cartesian_product)
of all their values. For correlated values (classic table tests), use
a single struct-typed parameter - see
[how to write parametrized tests](./how-to.md#how-to-write-parametrized-tests).

If a test declares a parameter with no matching `CasesXxx` method, or
the types mismatch, Testo reports an informative error before running
any tests. The [errors example](../examples/06_errors/main_test.go)
demonstrates these messages (that example fails on purpose).

## Suite Hooks

One day the whole bakery stopped producing pastry. It turned out
somebody had switched off the oven. The suite should manage the oven
itself.

Suites can define these hooks as methods:

- `BeforeAll(T)` - called _once before_ all tests. Its `T` refers to
  the top-level test, i.e. `Test/Bakery`.
- `BeforeEach(T)` - called _before each_ test, with that test's `T`.
- `AfterEach(T)` - called _after each_ test, before `t.Cleanup`
  callbacks, with that test's `T`.
- `AfterAll(T)` - called _once after_ all tests, including parallel
  ones. Its `T` refers to the top-level test.

We need the oven on once before all tests and off once after them:

```go
func (Bakery) BeforeAll(t T) {
	if err := Oven(true); err != nil {
		t.Fatalf("failed to turn the oven on: %v", err)
	}
}

func (Bakery) AfterAll(t T) {
	if err := Oven(false); err != nil {
		t.Errorf("failed to turn the oven off: %v", err)
	}
}
```

If `BeforeAll` fails, the suite's tests do not run at all.
For hook behavior with parallel sub-tests and panics, see
[how to write parallel tests](./how-to.md#how-to-write-parallel-tests)
and the [technical overview](./technical-overview.md#panics).

## Fixtures

The test currently creates a pastry and remembers to clean it up.
With more resources this gets repetitive, so we move the
create-and-cleanup logic out of the test.

`T` is our own type, which means we can add methods to it. A method
that creates a resource and schedules its cleanup is a fixture:

```go
// Bake is a fixture: it bakes a pastry and
// schedules the cleanup for the end of the test.
func (t T) Bake(name string) (Pastry, bool) {
	pastry, ok := Bake(name)

	if ok {
		t.Cleanup(func() { Eat(pastry) })
	}

	return pastry, ok
}
```

The test no longer deals with cleanup:

```go
func (Bakery) TestBake(t T, p struct{ Dessert string }) {
	pastry, ok := t.Bake(p.Dessert)
	if !ok {
		t.Fatalf("the bakery must know how to bake %s", p.Dessert)
	}

	if !pastry.Tasty {
		t.Errorf("%s must be tasty", p.Dessert)
	}
}
```

A fixture in Testo is an ordinary method. There is nothing to
register, and every test of the suite can use it.

## Steps (Sub-tests)

Right now a failure tells us *which* test failed, but not *which
check* inside it. We can structure the test into named steps using
sub-tests.

Sub-tests are started with the `testo.Run` function (this is the one
place where Testo differs from `t.Run`). Replace `TestBake` with:

```go
func (Bakery) TestBake(t T, p struct{ Dessert string }) {
	pastry, ok := t.Bake(p.Dessert)

	testo.Run(t, "check the bakery can bake it", func(t T) {
		if !ok {
			t.Fatalf("the bakery must know how to bake %s", p.Dessert)
		}
	})

	testo.Run(t, "check it is tasty", func(t T) {
		if !pastry.Tasty {
			t.Errorf("%s must be tasty", p.Dessert)
		}
	})
}
```

```txt
=== RUN   Test/Bakery/testo!/TestBake
=== RUN   Test/Bakery/testo!/TestBake/check_the_bakery_can_bake_it
=== RUN   Test/Bakery/testo!/TestBake/check_it_is_tasty
...
--- PASS: Test/Bakery/testo!/TestBake (0.00s)
    --- PASS: Test/Bakery/testo!/TestBake/check_the_bakery_can_bake_it (0.00s)
    --- PASS: Test/Bakery/testo!/TestBake/check_it_is_tasty (0.00s)
```

> [!WARNING]
> `t.Fatal` inside a sub-test stops only that sub-test, not the outer
> test. That is standard `go test` behavior. If a failed step must stop the
> whole test, check the sub-test result in the outer test, or use a
> plugin that propagates the failure, such as `allure.Step` from
> [testo-allure](https://github.com/ozontech/testo-allure).

Plugins get their own hooks around every sub-test:
`BeforeEachSub` and `AfterEachSub`.
Reporting plugins use sub-tests as steps - with
[testo-allure](https://github.com/ozontech/testo-allure) installed,
each `testo.Run` above becomes a step in the Allure report.

## Running Tests Without Suites

Suites are optional. You have already seen `testo.RunTest`:

```go
func TestFoo(t *testing.T) {
	testo.RunTest(t, func(t T) {
		t.Log("Hello from Testo!")
	})
}
```

And to run several tests from a single test function, there is the
`testo.Test` adapter for `t.Run`:

```go
func TestFoo(t *testing.T) {
	t.Run("FirstTest", testo.Test(func(t T) {
		t.Log("1!")
	}))

	t.Run("SecondTest", testo.Test(func(t T) {
		t.Log("2!")
	}))
}
```

## The Finished Test File

The complete `main_test.go` we built in this tutorial:

```go
package main

import (
	"testing"
	"time"

	"github.com/ozontech/testo"
	"github.com/ozontech/testo/testoplugin"
)

type PluginTimer struct {
	*testo.T

	start time.Time
}

func (p *PluginTimer) Plugin(testoplugin.Plugin, ...testoplugin.Option) (spec testoplugin.Spec) {
	spec.Hooks.BeforeEach.Func = func() {
		p.start = time.Now()
	}

	spec.Hooks.AfterEach.Func = func() {
		p.Logf("test %q took %s", p.Name(), time.Since(p.start))
	}

	return spec
}

type T struct {
	*testo.T
	*PluginTimer
}

// Bake is a fixture: it bakes a pastry and
// schedules the cleanup for the end of the test.
func (t T) Bake(name string) (Pastry, bool) {
	pastry, ok := Bake(name)

	if ok {
		t.Cleanup(func() { Eat(pastry) })
	}

	return pastry, ok
}

type Bakery struct{ testo.Suite[T] }

func (Bakery) BeforeAll(t T) {
	if err := Oven(true); err != nil {
		t.Fatalf("failed to turn the oven on: %v", err)
	}
}

func (Bakery) AfterAll(t T) {
	if err := Oven(false); err != nil {
		t.Errorf("failed to turn the oven off: %v", err)
	}
}

func (Bakery) CasesDessert() []string {
	return []string{"honey cake", "tiramisu"}
}

func (Bakery) TestBake(t T, p struct{ Dessert string }) {
	pastry, ok := t.Bake(p.Dessert)

	testo.Run(t, "check the bakery can bake it", func(t T) {
		if !ok {
			t.Fatalf("the bakery must know how to bake %s", p.Dessert)
		}
	})

	testo.Run(t, "check it is tasty", func(t T) {
		if !pastry.Tasty {
			t.Errorf("%s must be tasty", p.Dessert)
		}
	})
}

func Test(t *testing.T) {
	testo.RunSuite(t, new(Bakery))
}
```

## Next Steps

- Learn [how to use various Testo features](./how-to.md) - filtering
  tests, plugin options, persistent cache, project structure and more.
- Browse the [examples](../examples) - each is a small runnable
  project with its expected output.
- Read the [technical overview](./technical-overview.md) for the full
  lifecycle of a suite run.
- Migrating from testify or allure-go? See the
  [migration guide](./migration.md).
- View the [API documentation](https://pkg.go.dev/github.com/ozontech/testo).
