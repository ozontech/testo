# How to use Testo features

Snippets on this page are fragments: they assume a project set up as
in the [tutorial](./tutorial.md), with a suite and a `T` type already
defined.

## How to write parametrized tests

Parametrized tests are defined as regular tests with a second argument:

```go
func (*Suite) TestFoo(t T, p struct{ Name string; Age int }) {
    t.Logf("Using name=%q and age=%d", p.Name, p.Age)
}
```

To define all possible parameter values create a special `CasesXxx` method in a suite:

```go
func (*Suite) CasesName() []string {
    return []string{"John", "Joe"}
}

func (*Suite) CasesAge() []int {
    return []int{18, 60, 6}
}
```

> [!TIP]
> `CasesXxx` are invoked *after* the `BeforeAll` hook.

Each parameter field must have a matching `CasesXxx` method.

Given that, test `TestFoo` will be invoked with all possible combinations of names and ages:

```txt
TestFoo  with p = {Name: "John", Age: 18}
TestFoo  with p = {Name: "John", Age: 60}
TestFoo  with p = {Name: "John", Age: 6}
TestFoo  with p = {Name: "Joe",  Age: 18}
TestFoo  with p = {Name: "Joe",  Age: 60}
TestFoo  with p = {Name: "Joe",  Age: 6}
```

Parameter values do not appear in test names.
In `go test -v` output the runs are named `TestFoo`, `TestFoo#01`, `TestFoo#02` and so on,
following the standard `go test` convention for repeated names.
Log the parameters at the start of the test, so a failing `TestFoo#03`
identifies its case - or let a
[ten-line plugin](./plugins.md#reading-test-metadata) do it for every
test automatically.

If a `CasesXxx` method returns an empty slice, Testo logs a warning
(visible with `go test -v`):

```txt
main_test.go:15: testo: warning: (*main.Suite).CasesName returned empty slice, (*main.Suite).TestFoo will not run
```

To make this warning fatal, enable
[strict mode](#how-to-enable-strict-mode).

### Table tests (correlated parameters)

Separate parameters always produce the Cartesian product of their values.
When values are correlated - a classic table test, where each case is one
row with its own input and expected output - use a single struct-typed
parameter instead:

```go
type Case struct {
    Input string
    Want  int
}

func (*Suite) CasesCase() []Case {
    return []Case{
        {Input: "one", Want: 1},
        {Input: "two", Want: 2},
    }
}

func (*Suite) TestParse(t T, p struct{ Case Case }) {
    if got := Parse(p.Case.Input); got != p.Case.Want {
        t.Errorf("Parse(%q) = %d, want %d", p.Case.Input, got, p.Case.Want)
    }
}
```

The test runs once per element of the slice, with no cross-combination.

## How to write parallel tests

Mark a test as parallel with the standard `t.Parallel`:

```go
func (*Suite) TestFoo(t T) {
    t.Parallel()

    // your test here
}
```

Standard `go test` flags such as `-parallel`, `-count` and `-timeout`
apply unchanged - Testo tests are regular Go tests underneath.
With `-count=N`, `BeforeAll` and `AfterAll` run once per iteration.

### Hooks and parallel sub-tests

`AfterEach` and `AfterAll` still run for parallel tests, but
`AfterEach` is deferred to the end of the test body. If the test has
parallel sub-tests, the hook runs **before** they finish:

```go
func (*Suite) Test(t T) {
    testo.Run(t, "my sub-test", func(t T) {
        testo.Run(t, "sub-sub-test 1", func(t T) {
            t.Parallel()

            time.Sleep(time.Second)
        })

        testo.Run(t, "sub-sub-test 2", func(t T) {
            t.Parallel()

            time.Sleep(time.Second)
        })

        // AfterEach would run here.
        // But these parallel sub-tests will run later, not now.
    })
}
```

If your teardown must run after all parallel sub-tests finish,
register it with `t.Cleanup` inside `BeforeEach` instead of using the `AfterEach` hook -
cleanups run after all parallel sub-tests of the test are done:

```go
func (*Suite) BeforeEach(t T) {
    t.Cleanup(func() {
        // Teardown logic here.
        // Runs after the test AND all its parallel sub-tests finish.
    })
}
```

## How to use plugin options

Plugins can accept options to alter their behavior.

Passing to `testo.RunSuite`:

```go
func Test(t *testing.T) {
    testo.RunSuite(
        t,
        new(Suite),
        myplugin.SomeOption(),
        otherplugin.OtherOption(42),
    )

    testo.RunSuite(
        t,
        new(OtherSuite),
        anotherplugin.Option("..."),
    )
}
```

Using `testo.Options`:

```go
var _ = testo.Options(
    myplugin.SomeOption(),
    otherplugin.OtherOption(42),
)

func Test(t *testing.T) {
    // options are automatically passed to these RunSuite calls.
    testo.RunSuite(t, new(Suite))
    testo.RunSuite(t, new(OtherSuite))
}
```

Passing to `testo.Run`:

```go
func (s *Suite) TestFoo(t T) {
    testo.Run(t, "my sub-test", func(t T) {
        // ...
    }, myplugin.SomeOption())
}
```

> [!NOTE]
> The plugin author decides if an option is passed to inner
> sub-tests. You can force this with the `.Propagate` field,
> but usually you should not.

## How to use persistent cache

`testocache` stores key-value data between `go test` runs:

```go
import "github.com/ozontech/testo/testocache"

err := testocache.Set("token", []byte("abc"))
value, err := testocache.Get("token")
```

By default the cache lives in `.testo_cache` in the test working
directory:

```bash
# change the directory for one package
go test ./path/to/package -cache.dir /tmp/my-testo-cache

# for ./... use the env var, so packages that don't import
# testocache don't fail on an unknown flag
TESTO_CACHE_DIR=/tmp/my-testo-cache go test ./...

# disable the cache entirely
TESTO_CACHE_DISABLE=true go test ./...
```

> [!NOTE]
> Cache flags use the `-cache.` prefix, not `-testo.` -
> there is no `-testo.cache.dir`.

For plugin state, prefer a namespace. It is isolated from other
namespaces and from the package-level functions, and has the same
`Get`, `Set`, `Keys` and `Remove` methods:

```go
var cache = testocache.Namespace("myplugin")
```

Behavior details:

- `Get` and `Remove` return `testocache.ErrNotFound` for missing keys.
- All operations return `testocache.ErrDisabled` when the cache is off.
- `Keys` accepts the same glob syntax as `path.Match`.
- Writes are atomic (temporary file + `os.Rename`); malformed entries
  are treated as missing.
- Operations are synchronized within one process. There is no locking
  between separate `go test` processes sharing a cache directory.

> [!NOTE]
> `go test` caches successful test results, and a cached pass skips
> execution - so `testocache` state will not refresh. Pass `-count=1`
> to force a run.

## How to structure tests

Testo does not enforce any particular file structure.
Here is one pattern we find useful - standalone suite packages:

```txt
go.mod
go.sum
tests
├── suite
│   └── suitefoo
│       ├── suite.go
│       └── t.go
├── suite_test.go
└── testcommon
    └── plugin.go
```

Contents of `tests/testcommon/plugin.go`:

```go
package testcommon

import (
    "github.com/ozontech/testo"
    "github.com/ozontech/testo/testoplugin"
)

// A plugin shared by all tests.
// Define it even if you have no plugins yet. When you add
// a plugin later, you will change only this file.
type PluginCommon struct {
    *testo.T
}

// This method implements the Plugin interface.
func (*PluginCommon) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
    return testoplugin.Spec{}
}
```

Contents of `tests/suite/suitefoo/t.go`:

```go
package suitefoo

import (
    "example/tests/testcommon"

    "github.com/ozontech/testo"
)

// T for this suite. With several suites,
// hoist it into a shared package.
type T struct {
    *testo.T
    *testcommon.PluginCommon
}
```

Contents of `tests/suite/suitefoo/suite.go`:

```go
package suitefoo

import "github.com/ozontech/testo"

// The test logic for this suite goes here.

type Suite struct{ testo.Suite[T] }

func (*Suite) TestItWorks(t T) {
    t.Log("works!")
}
```

Contents of `tests/suite_test.go`:

```go
//go:build integration

package tests

import (
    "testing"

    "example/tests/suite/suitefoo"

    "github.com/ozontech/testo"
)

// All suites are started here.
// One file shows at a glance which suites run,
// and the build tag ("integration") is declared once.

func TestSuiteFoo(t *testing.T) {
    testo.RunSuite(t, new(suitefoo.Suite))
}
```

## How to run and skip specific tests

Testo supports the standard `go test` flags, such as `-run` and `-skip`.

See `go help testflag` for the full flag reference.

To make hooks work correctly with parallel tests, Testo inserts a hidden
`testo!` level into every suite. The full name of a suite test is:

```txt
TestFunc/SuiteName/testo!/TestMethod[/sub-test...]
```

> [!WARNING]
> A `-run` pattern without the `testo!` segment, such as
> `-run 'Test/MySuite/TestFoo'`, matches **zero tests** - and `go test`
> still exits 0 (at best you get an easy-to-miss `[no tests to run]`
> note). Suite hooks (`BeforeAll`/`AfterAll`) still run even when zero
> tests match.
>
> Either include the segment (`-run 'Test/MySuite/testo!/TestFoo'`)
> or, better, use the `-testo.m` flag below. In VS Code, the
> [Testo extension](../vscode-extension) generates correct commands.

Testo provides its own flag `-testo.m regexp` to select suite tests by
method name, without worrying about the `testo!` segment.

For example, given the following suite:

```go
type MySuite struct { testo.Suite[T] }

func (MySuite) TestFoo(t T) {
    // ...
}

func (MySuite) TestBar(t T) {
    // ...
}

func Test(t *testing.T) {
    testo.RunSuite(t, new(MySuite))
}
```

To run only `TestFoo`:

```shell
go test . -run 'Test/MySuite' -testo.m TestFoo
```

Here `-run 'Test/MySuite'` selects the suite and `-testo.m TestFoo`
selects the method inside it.

> [!NOTE]
> `t.Name()` returns the name without `testo!`, e.g. `Test/MySuite/TestFoo`.
> The real `testing.T` name (the one in `go test -v` output and in `-run`
> patterns) still contains `testo!`. Remember this when you parse test
> output or build `-run` patterns.

## How to enable strict mode

Strict mode turns every Testo warning into a fatal error.
Currently two warnings exist - a suite with no tests, and a
`CasesXxx` method returning an
[empty slice](#how-to-write-parametrized-tests) - and the list
may grow:

```bash
# one package
go test ./path/to/package -testo.strict

# for ./... use the env var - packages that don't import
# Testo would fail on the unknown flag
TESTO_STRICT=true go test ./...
```

> [!TIP]
> Flags have higher priority than environment variables.

## How to annotate tests

Annotations attach static plugin options to a specific test, so
plugins can see them before the test runs - for example to plan
retries or add report labels.

Use `testo.For` for regular tests and `testo.ForEach` for parametrized ones:

```go
var _ = testo.For(MySuite.TestFoo, myplugin.WithRetry())

func (MySuite) TestFoo(t T) {
    // ...
}

var _ = testo.ForEach(MySuite.TestBar, myplugin.WithRetry())

func (MySuite) TestBar(t T, p struct{ N int }) {
    // ...
}
```

Multiple annotation calls for the same test append options.

An option is just a `testoplugin.Option` value wrapping any type your
plugin knows how to interpret:

```go
func WithRetry() testoplugin.Option {
    return testoplugin.Option{Value: retryOption{}}
}
```

Plugins see annotations in two places: during planning, via
`PlannedTest.Annotations()` in `Plan.Prepare`, and per test, merged
into the `options` argument of the `Plugin` method. In both cases the
plugin type-asserts the `Value` field.

See [options in the plugins guide](./plugins.md#options),
the [annotations example](../examples/07_annotations/main_test.go)
(its `plugin.go` defines the options)
and [API documentation](https://pkg.go.dev/github.com/ozontech/testo#For).

## How to integrate with CI

Testo output is standard `go test` output. `go test -json`, `test2json`,
[gotestsum](https://github.com/gotestyourself/gotestsum) and similar tools
work unchanged.

Notes:

- Read [how to run and skip specific tests](#how-to-run-and-skip-specific-tests).
- The hidden `testo!` node appears in reports as an extra nesting level
  (e.g. JUnit converters render it as an empty intermediate node).
- For Allure reports, use the [testo-allure plugin](https://github.com/ozontech/testo-allure).
- For rerunning only failed tests (flaky-test handling), see the
  [rerun plugin](https://github.com/ozontech/testo-toppings/tree/main/rerun).

## How to run sub-suites

There is a `testo.RunSubSuite` function for that:

```go
type OuterSuite struct{ testo.Suite[T] }

func (OuterSuite) Test(t T) {
	testo.RunSubSuite(t, new(InnerSuite))
}

type InnerSuite struct{ testo.Suite[T] }

func (InnerSuite) Test(t T) {
	t.Log("Hello from sub-suite!")
}
```

See the [sub-suites example](../examples/08_subsuites/main_test.go).

> [!WARNING]
> Running a suite inside itself causes an infinite loop.
> Keep sub-suite nesting acyclic.
