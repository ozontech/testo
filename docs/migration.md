# Migration guide

How to move an existing test base to Testo - from
[testify](#migrating-from-testify) or from
[allure-go](#migrating-from-allure-go).

Both migrations can be done incrementally: Testo tests are regular
`go test` tests, so old and new suites run side by side in the same
package with no interference. Migrate one suite at a time.

## Migrating from testify

### Suites and hooks

| testify | Testo |
| :------ | :---- |
| `suite.Suite` (embedded) | `testo.Suite[T]` (embedded) |
| `suite.Run(t, new(S))` | `testo.RunSuite(t, new(S))` |
| `func (s *S) SetupSuite()` | `func (*S) BeforeAll(t T)` |
| `func (s *S) SetupTest()` | `func (*S) BeforeEach(t T)` |
| `func (s *S) TearDownTest()` | `func (*S) AfterEach(t T)` |
| `func (s *S) TearDownSuite()` | `func (*S) AfterAll(t T)` |
| `func (s *S) TestFoo()` | `func (*S) TestFoo(t T)` |
| `s.T()` | `t` (passed to every method) |
| `s.Run(name, func())` | `testo.Run(t, name, func(t T))` |
| `func (s *S) BeforeTest(suiteName, testName string)` | `BeforeEach(t T)` + `t.Name()` (same for `AfterTest`) |

Two notes on the hook mapping. `t.Name()` returns the full path
(`Test/MySuite/TestFoo`), not the bare method name - compare with
`strings.HasSuffix`, not `==`. And if a suite uses both `SetupTest`
and `BeforeTest`, merge them into one `BeforeEach` (testify runs
`SetupTest` first; keep that order).
| `SetupSubTest()` / `TearDownSubTest()` | `BeforeEachSub`/`AfterEachSub` plugin hooks (no suite-level equivalent) |
| `go test -run TestSuite -testify.m TestFoo` | `go test -run TestSuite -testo.m TestFoo` |

One flag caveat: testify method filters like `-run 'TestMySuite/TestFoo'`
stop matching after migration, because Testo inserts a hidden `testo!`
level into test names. Update CI filters to use `-testo.m`. See
[how to run and skip specific tests](./how-to.md#how-to-run-and-skip-specific-tests).

`T` here is your own type built on `*testo.T` - in the simplest case
just an alias:

```go
type T = *testo.T
```

See the [tutorial](./tutorial.md) for defining a `T` with plugins.

### Assertions

testify's `require` and `assert` packages work with Testo unchanged,
because `testo.T` implements `testing.TB`:

```go
// testify/suite:
func (s *MySuite) TestFoo() {
    s.Require().NoError(err)
    s.Equal(want, got) // shorthand from the embedded suite.Suite
}

// Testo:
func (*MySuite) TestFoo(t T) {
    require.NoError(t, err)
    assert.Equal(t, want, got)
}
```

The `s.Require().X(...)` and shorthand `s.X(...)` forms both become
plain `require.X(t, ...)` / `assert.X(t, ...)` calls.

Testo only replaces testify's `suite` package. Keep using `require`
and `assert`.

### Suite state

testify suites keep per-test state in suite struct fields.
A Testo suite is also a single instance, so shared fields still work.
But writes to shared fields from parallel tests are a data race -
run migrated packages with `-race` before enabling `t.Parallel()`.
For per-test state, prefer [fixtures](./tutorial.md#fixtures)
(methods on your `T`) or plugin fields. Both get a fresh instance
for each test and are safe with parallel tests.

### Parallelism

testify's `suite` does not support `t.Parallel()`
([issue #934](https://github.com/stretchr/testify/issues/934)).
With Testo, call `t.Parallel()` in any test - hooks handle parallel
tests correctly. See
[how to write parallel tests](./how-to.md#how-to-write-parallel-tests).

## Migrating from allure-go

Testo [replaces allure-go at Ozon](../COMPARISON.md#why-we-moved-off-allure-go).
Reporting is now a separate plugin, not part of the framework.
So you need **two** pieces: `github.com/ozontech/testo` and the
[testo-allure](https://github.com/ozontech/testo-allure) plugin.

```go
import (
    "github.com/ozontech/testo"

    allure "github.com/ozontech/testo-allure"
)

type T struct {
    *testo.T
    *allure.PluginAllure
}
```

### Concept mapping

| allure-go | Testo (+ testo-allure) |
| :-------- | :--------------------- |
| `suite.Suite` | `testo.Suite[T]` |
| `suite.RunSuite(t, new(S))` (or `runner.NewSuiteRunner(...)`) | `testo.RunSuite(t, new(S))` |
| `BeforeAll(t provider.T)` | `BeforeAll(t T)` (same for the other hooks) |
| `func (s *S) TestFoo(t provider.T)` | `func (*S) TestFoo(t T)` |
| `t.WithNewStep("name", func(ctx provider.StepCtx) {...})` | `allure.Step(t, "name", func(t T) {...})` |
| `t.NewStep("name")` | `allure.Step(t, "name", func(T) {})` |
| `t.Title`, `t.Epic`, `t.Feature`, `t.Story`, `t.Tags`, `t.Severity`, `t.Owner`, `t.ID` | same-named methods added to `T` by `PluginAllure` |
| `t.Link(...)`, `t.TmsLink(...)`, `t.TmsLinks(...)` | `t.Links(...)` with the `allure.NewLink`/`allure.TMS`/`allure.Issue` constructors |
| `t.WithNewAttachment(name, mimeType, content)`, `t.WithAttachments(...)` | `t.Attach(name, allure.Bytes(...))` or `t.Attach(name, allure.File(...))` |
| table tests (`ParametrizedSuite` / `TableTestXxx` methods) | single struct-typed parameter, see [table tests](./how-to.md#table-tests-correlated-parameters) |
| `t.XSkip()` | `t.XFail()` via the [xfail plugin](https://github.com/ozontech/testo-toppings/tree/main/xfail) from testo-toppings (semantics differ slightly - check its README) |

Other labels (`Description`, `Labels`, `Layer`, `Lead`, `Stage`,
`WithParameters`) map to the `Description`, `Labels` and `Parameters`
methods of `PluginAllure`.

A minimal setup to start a proof of concept:

```go
func Test(t *testing.T) {
    testo.RunSuite(t, new(Bakery), allure.WithOutputDir("allure-results"))
}
```

### Steps

Steps are sub-tests. Two ways to create them:

- `testo.Run(t, "name", func(t T) {...})` - the framework primitive.
  `t.Fatal` inside stops only the sub-test, not the outer test.
- `allure.Step(t, "name", func(t T) {...})` - the testo-allure
  helper. Same as `testo.Run`, but a fatal failure propagates to the
  parent test and stops it - matching `WithNewStep` semantics from
  allure-go.

For code full of `WithNewStep`, use `allure.Step` - same semantics.
Nested steps (`sCtx.WithNewStep`, `sCtx.NewStep`) become nested
`allure.Step(t, ...)` calls. Step assertions (`sCtx.Assert()`,
`sCtx.Require()`) become `t.Assert()` / `t.Require()`, which
`PluginAllure` adds to `T`.

For the full testo-allure API (output directory, labels, attachments
and so on), see the
[testo-allure documentation](https://github.com/ozontech/testo-allure).
