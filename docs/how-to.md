# How to

Learn how to use the features of Testo.

## How to write parametrized tests

Parametrized tests are defined as regular tests with a second argument:

```go
func (*Suite) TestFoo(t *testo.T, p struct{ Name string; Age int }) {
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
> `CasesXxx` are invoked *after* `BeforeAll` hook.

Field names used in a `struct{ Name string; Age int}` must be equal to existing `CasesXxx` functions.

Given that, test `TestFoo` will be invoked with all possible combinations of names and ages:

```python
TestFoo(name=John, age=18)
TestFoo(name=John, age=60)
TestFoo(name=John, age=6)
TestFoo(name=Joe, age=18)
TestFoo(name=Joe, age=60)
TestFoo(name=Joe, age=6)
```

If for at least one required parameter function `CasesXxx`
returns zero values Testo will log a warning with similar message:

```txt
main_test.go:15: testo: (*main.Suite).CasesName provides zero values, (*main.Suite).TestFoo won't run
```

To turn this log into fatal error and do not proceed with further execution
pass flag `-testo.strict` to the `go test` command invocation:

```bash
go test ./... -testo.strict
```

> [!TIP]
> You can also set `TESTO_STRICT` environment variable to `true`
> for the same effect.
>
> Flags have higher priority than environment variables.

## How to write parallel tests

You can use your regular `t.Parallel` method to mark a test as parallel.

```go
func (*Suite) TestFoo(t *testo.T) {
    t.Parallel()

    // your test here
}
```

You can expect all `AfterEach` and `AfterAll` hooks to execute at the end of each test properly.

Please note, that `AfterEach` (only applies to it) hook is deferred to run at the end of the test.
If that test has sub-tests marked as parallel,
this hook will run BEFORE those sub-tests are finished.

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

Unless you need to run sub-tests during this hook,
it is recommended to use t.Cleanup during BeforeEach.

```go
func (*Suite) BeforeEach(t T) {
    t.Cleanup(t.afterEach)
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
func (s *Suite) TestFoo(t *testo.T) {
    testo.Run(t, "my sub-test", func(t *testo.T) {
        // ...
    }, myplugin.SomeOption())
}
```

> Options can be propagated to inner sub-tests, but it's up to a plugin author
> to decide whether an option should be propagated or not.
>
> If needed, you can change propagation for certain options
> by setting `.Propagate` field to `false` (disable) or `true` (enable).
> However, it is highly discouraged to do so.

## How to use persistent cache

`testocache` stores key-value data between `go test` runs.
By default it uses `.testo_cache` in the test working directory.

```bash
go test ./path/to/package -cache.dir /tmp/my-testo-cache
```

When testing multiple packages with `./...`, use the environment variable so
packages that do not import `testocache` do not receive an unknown test flag:

```bash
TESTO_CACHE_DIR=/tmp/my-testo-cache go test ./...
```

```bash
TESTO_CACHE_DISABLE=true go test ./...
```

For plugin state, prefer a namespace:

```go
var cache = testocache.Namespace("myplugin")
```

Namespaces are isolated from each other and from package-level
`testocache.Get`, `Set`, `Keys`, and `Remove`.
The scoped cache provides the same `Get`, `Set`, `Keys`, and `Remove` methods.
`Keys` accepts the same glob syntax as `path.Match`.

If cache is disabled, operations return `testocache.ErrDisabled`.
If a key is missing, `Get` and `Remove` return `testocache.ErrNotFound`.
Cache writes use a temporary file followed by `os.Rename`; atomic replacement
follows the guarantees of `os.Rename` on the host platform.
Malformed entries and entries whose stored key does not match the requested key
are treated as missing. Concurrent operations are synchronized inside one test
process; Testo does not provide cross-process locking for multiple `go test`
processes sharing the same cache directory.

## How to structure tests

Testo does not enforce any particular file structure to work.
However, some patterns are proved to be useful.

### Standalone suites

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

// Here we define a global plugin common for all tests.
//
// It is highly advised to make it even if you don't use any plugins yet,
// as in the future it won't require any further changes in other files if you
// decide to add plugins.
type PluginCommon struct {
    *testo.T
}

// This method implements Plugin interface.
func (*PluginCommon) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
    return testolpugin.Spec{}
}
```

Contents of `tests/suite/suitefoo/t.go`:

```go
package suitefoo

import "example/tests/testcommon"

// Global (common) T used by all suites.
type T struct {
    *testo.T
    *testcommon.PluginCommon
}
```

Contents of `tests/suite/suitefoo/suite.go`:

```go
package suitefoo

import "github.com/ozontech/testo"

// An actual test logic for this suite goes here.

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

// In this file we actually run our suites.
// The reason for starting tests here is that
// single file makes it easier to see what suites will we run or skip.
//
// Moreover, if you add some build tag, like we do here ("integration" on the top),
// specifying it only once makes it less typo-prone.

func TestSuiteFoo(t *testing.T) {
    testo.RunSuite(t, new(suitefoo.Suite))
}
```

## How to run and skip specific tests

Testo works with default go test flags, such as `-run` and `-skip`.

> See `go help testflag` for detailed flags description.

Testo also provides its own flag: `-testo.m regexp` to run specific suite tests.

For example, given the following suite:

```go
type MySuite struct { testo.Suite[T] }

func (MySuite) TestFoo(t T) {
    // ...
}

func (MySuite) TestBar(t T) {
    // ...
}
```

We can run only `TestFoo` like that:

```shell
go test . -run ./MySuite -testo.m TestFoo
```

> [!TIP]
> See also [Visual Studio Code extension](../vscode-extension) which does just that for you.

## How to run sub-suites

There a `testo.RunSubSuite` function for that:

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

> [!WARNING]
> Running the same suite as sub-suite may cause infinite loop.
