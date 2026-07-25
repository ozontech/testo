# Writing plugins

Everything a plugin can do, with a compiling example for each part.
This page assumes you finished the [tutorial](./tutorial.md), where a
first plugin (the timer) is built step by step.

A plugin is a struct that users embed into their `T`:

```go
type T struct {
    *testo.T
    *PluginShout
}
```

Testo collects the plugins from the `T` type, creates a fresh
instance of each for every test and sub-test, and asks each one what
it wants to do by calling its `Plugin` method:

```go
func (p *PluginShout) Plugin(parent testoplugin.Plugin, options ...testoplugin.Option) testoplugin.Spec
```

The returned `testoplugin.Spec` has three parts, all optional:

| Part | What it does |
| :--- | :----------- |
| `Hooks` | run code around suites, tests and sub-tests |
| `Overrides` | wrap built-in `T` methods (`Log`, `Fail`, ...) |
| `Plan` | filter, reorder or duplicate the tests before the run |

A plugin usually embeds `*testo.T`. Testo fills it with the same `T`
as the current test, so the plugin can log, fail, or inspect the test
it runs in. Pin the interface at compile time:

```go
var _ testoplugin.Plugin = (*PluginShout)(nil)
```

The `parent` argument is the plugin instance of the enclosing scope:
the suite root for a test, the test for a sub-test. Only at the suite
root itself is it a typed-nil pointer, so assert and nil-check the
pointer, not the interface:

```go
prev, _ := parent.(*PluginShout)
if prev != nil {
    // not the suite root: prev belongs to the enclosing test or suite
}
```

## Hooks

Six hooks: the four suite hooks plus `BeforeEachSub`/`AfterEachSub`,
which have no suite-level counterpart:

```go
spec.Hooks.BeforeEach = testoplugin.Hook{
    Priority: testoplugin.TryFirst,
    Func: func() {
        p.Logf("starting %s", p.Name())
    },
}
```

`BeforeAll`/`AfterAll` run once per suite, `BeforeEach`/`AfterEach`
around each test, `BeforeEachSub`/`AfterEachSub` around each
sub-test. `Priority` is an `int` that orders hooks across plugins:
lower values run earlier, any value works, `TryFirst` and `TryLast`
are the extremes, zero keeps declaration order.

`AfterEach` and `AfterEachSub` are deferred, so with parallel
sub-tests they run before those sub-tests finish - same caveat as
[suite hooks](./how-to.md#hooks-and-parallel-sub-tests).

## Overrides

Overrides wrap built-in `T` methods in middleware style: your
function receives the next implementation and returns a replacement.

```go
spec.Overrides.Log = func(next testoplugin.FuncLog) testoplugin.FuncLog {
    return func(args ...any) {
        p.Helper() // keep the caller's file:line in the output
        next(append([]any{"[quiet]"}, args...)...)
    }
}
```

Call `t.Helper()` inside the wrapper, or every log line will point
at the wrapper's own file and line.

Overridable methods: `Log`, `Error`, `Fatal`, `Fail`, `FailNow`,
`Failed`, `Skip`, `SkipNow`, `Skipped`, `Parallel`, `Cleanup`,
`Context`, `Deadline`, `TempDir`, `Setenv`, `Chdir`.
Each method `X` has a matching `testoplugin.FuncX` signature type.

Methods call each other underneath - `Error` is `Log` + `Fail`,
`Fatal` is `Log` + `FailNow` - so overriding `Log` also affects
`Error` and `Fatal`. `Overrides.Priority` orders stacks across
plugins: the lowest priority becomes the outermost wrapper, so its
code runs first. For a real example, the
[xfail plugin](https://github.com/ozontech/testo-toppings/tree/main/xfail)
overrides `Fail` and `FailNow` to turn expected failures into skips.

## Planning tests

`Plan.Prepare` receives the collected tests before the run and may
filter, reorder or duplicate them in place:

```go
import "github.com/ozontech/testo/testoreflect"

spec.Plan.Prepare = func(suite testoreflect.SuiteInfo, tests *[]testoplugin.PlannedTest) {
    slices.Reverse(*tests)
}
```

Each `PlannedTest` exposes `Annotations()` - the options attached to
that test with [`testo.For`](./how-to.md#how-to-annotate-tests) - and
`Info()`, which returns a `testoreflect.TestInfo` interface:

```go
name := t.Info().GetName()
```

`GetName()` returns the full run path without the `testo!` segment,
e.g. `Test/Suite/TestFoo#01` - match with `strings.HasSuffix`, not `==`.

Parametrized tests arrive already expanded, one `PlannedTest` per
case. The initial order is regular tests first (alphabetical), then
parametrized cases. One display quirk: after reordering, `go test`
reassigns the `#01` suffixes by run order, while `t.Name()` keeps the
original case index.

The [rerun plugin](https://github.com/ozontech/testo-toppings/tree/main/rerun)
uses `Prepare` to drop every test that passed in the previous run.

## Adding methods to T

Any method on the plugin becomes a method on `T` through embedding.
If methods are all your plugin does, you don't even need a `Plugin`
method:

```go
type PluginGreet struct{ *testo.T }

func (g *PluginGreet) Greet() { g.Log("hello") }
```

```go
func (Suite) TestFoo(t T) {
    t.Greet()
}
```

## Options

An option is a `testoplugin.Option` wrapping any value your plugin
recognizes. Users pass options to `testo.RunSuite`, `testo.Run` or
`testo.Options` - see
[how to use plugin options](./how-to.md#how-to-use-plugin-options).
The common pattern is an unexported function type:

```go
type shoutOption func(*PluginShout)

func WithPrefix(prefix string) testoplugin.Option {
    return testoplugin.Option{
        Value:     shoutOption(func(p *PluginShout) { p.prefix = prefix }),
        Propagate: true, // pass to sub-tests too
    }
}
```

Consume options at the top of the `Plugin` method. All user-supplied
options come through the variadic argument (including per-test
[annotations](./how-to.md#how-to-annotate-tests)); ignore values that
are not yours:

```go
func (p *PluginShout) Plugin(_ testoplugin.Plugin, options ...testoplugin.Option) (spec testoplugin.Spec) {
    for _, o := range options {
        if o, ok := o.Value.(shoutOption); ok {
            o(p)
        }
    }
    // ...
    return spec
}
```

## Command line flags

Plugins register flags with the standard `flag` package at package
level. The `go test` machinery parses them like any other test flag:

```go
var flagShout = flag.Bool("shout.enabled", false, "print a banner before each test")
```

```bash
go test . -shout.enabled
```

Read the value inside `Plugin` or a hook, after flags are parsed.
Prefix flag names with your plugin name to avoid collisions. Real
examples: `-rerun.failed` in the rerun plugin, `-allure.dir` in
testo-allure.

Flags only exist in packages that import the plugin, so `go test
./... -shout.enabled` fails in packages that don't. Pair each flag
with an environment variable if your plugin must be configurable
across a whole repo (Testo does this with `TESTO_STRICT` and
`TESTO_CACHE_DIR`).

## Reading test metadata

`testo.Reflect` works on any `T` - including the plugin itself, via
its embedded `*testo.T`:

```go
spec.Hooks.BeforeEach.Func = func() {
    info, ok := testo.Reflect(p).Test.(testoreflect.ParametrizedTestInfo)
    if ok {
        p.Logf("params: %v", info.Params)
    }
}
```

That snippet answers "which parameters did `TestFoo#01` run with":
the plugin logs them for every parametrized test. The `Reflection` struct also carries the suite info, failure
kind and source, and panic details - which is how
[testo-allure](https://github.com/ozontech/testo-allure) fills its
reports. See the
[testoreflect reference](https://pkg.go.dev/github.com/ozontech/testo/testoreflect).

## Talking to other plugins

Plugins reference each other by embedding; Testo reuses one instance
per test across all references. See
[cross-plugin communication](./technical-overview.md#plugins).

## Plugins worth reading

- [examples/04_plugins](../examples/04_plugins/main_test.go) - hooks,
  overrides and planning in one small file.
- [examples/07_annotations](../examples/07_annotations/plugin.go) - a
  plugin that defines options and consumes annotations in `Plan`.
- [testo-toppings](https://github.com/ozontech/testo-toppings) - four
  small production plugins (rerun, xfail, parallel, async).
- [testo-allure](https://github.com/ozontech/testo-allure) - a large
  reporting plugin using every part of the `Spec`.
