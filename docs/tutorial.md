# Tutorial

## Create a New Go Project

```bash
mkdir testo-tutorial
cd testo-tutorial
go mod init testo-tutorial
```

Create `main.go`:

```go
package main

// Add returns sum of the given integers.
func Add(a, b int) int { return a + b }

func main() {}
```

## Writing Tests

Testo uses `testo.T` - a wrapper around `testing.T` with extended features.
All methods available in `testing.T` are also available in `testo.T` (except `t.Run`, more on that later in [Sub-tests chapter](#sub-tests)).
Also, `testo.T` implements the `testing.TB` interface.

**Important**: We'll create an alias to avoid repetition.

Create file `main_test.go`:

```go
package main

import (
    "testing"

    "github.com/ozontech/testo"
)

type T = *testo.T
```

Now we need a Suite. A Suite must "inherit" `testo.Suite[T]` by embedding it.

> [!NOTE]
> It's possible to run tests without suites, more on that in [later](#running-tests-without-suites).

```go
type Suite struct{ testo.Suite[T] }

// Tests in Suites are regular methods, following the same
// naming rules as regular tests in Go.
// That means they must have the "Test" prefix.
// They also must use the same type T as specified in `testo.Suite[T]`.

func (*Suite) TestAdd(t T) {
    if Add(2, 2) != 4 {
        t.Fatal("2 + 2 must equal 4")
    }
}
```

## Running Tests

To connect Testo with `go test` we must run it from a regular test:

```go
func Test(t *testing.T) {
    testo.RunSuite(t, new(Suite))
}
```

Now we can run tests:

```bash
go test . -v
```

```txt
=== RUN   Test
=== RUN   Test/Suite
=== RUN   Test/Suite/testo!
=== RUN   Test/Suite/testo!/TestAdd
--- PASS: Test (0.00s)
    --- PASS: Test/Suite (0.00s)
        --- PASS: Test/Suite/testo! (0.00s)
            --- PASS: Test/Suite/testo!/TestAdd (0.00s)
PASS
```

You may notice a special test `testo!` - Testo creates it "under the hood"
to guarantee correct work of hooks with parallel tests.

Don't worry too much about this, as its existence doesn't affect your tests.

## Suite Hooks

Suites can have the following hooks:

- `BeforeAll(T)` - called _once before_ all tests. The passed `T` refers to the top test, i.e. `Test/Suite`.
- `BeforeEach(T)` - called _before each_ test. The passed `T` refers to the same test that the hook runs before.
- `AfterEach(T)` - called _after each_ test, but before `t.Cleanup`. The passed `T` refers to the same test that the hook runs after.
- `AfterAll(T)` - called _once after_ all tests. The hook waits for all (parallel) tests to finish. The passed `T` refers to the top test, i.e. `Test/Suite`.

Hooks are defined as Suite methods:

```go
func (*Suite) BeforeEach(t T) {
    t.Logf("Starting: %s", t.Name())
}

func (*Suite) AfterEach(t T) {
    t.Logf("Finished: %s", t.Name())
}
```

<details>
<summary>Output:</summary>

```txt
=== RUN   Test
=== RUN   Test/Suite
=== RUN   Test/Suite/testo!
=== RUN   Test/Suite/testo!/Add
    main_test.go:20: Starting: Test/Suite/TestAdd
    main_test.go:28: Test/Suite/TestAdd
    main_test.go:24: Finished: Test/Suite/TestAdd
--- PASS: Test (0.00s)
    --- PASS: Test/Suite (0.00s)
        --- PASS: Test/Suite/testo! (0.00s)
            --- PASS: Test/Suite/testo!/TestAdd (0.00s)
PASS
```

</details>

## Parametrized Tests

Parametrized tests let you run the same tests with different input parameters.

Parametrized tests follow the same naming rules as regular tests, but take
a second argument after `T`.
This argument must be a struct containing the required parameters.

```go
func (*Suite) TestAddButParametrized(t T, p struct{ A, B int }) {
    if Add(p.A, p.B) != Add(p.B, p.A) {
        t.Errorf("%[1]d + %[2]d != %[2]d + %[1]d", p.A, p.B)
    }
}
```

We also need to declare the parameter values with which the test will run.
This is done by declaring special methods with names like `CasesXXX`, where
`XXX` is the parameter name.

```go
func (*Suite) CasesA() []int {
    return []int{1, 2, 3, 4, 5}
}

func (*Suite) CasesB() []int {
    return []int{11, 1000, 13}
}
```

Parametrized tests are called with the [Cartesian product](https://en.wikipedia.org/wiki/Cartesian_product)
of all values obtained from `CasesXXX` functions.
For the example above, this is 15 different pairs of values.

If a test specifies a parameter for which no corresponding `Cases` function is found,
Testo will print an informative error before running any tests.
The same applies to type mismatches.

See [error examples](../../examples/07_errors/main_test.go).

## Sub-tests

Running sub-tests is a bit different from the usual `t.Run` call.

Sub-tests must be started using the `testo.Run` function:

```go
func (*Suite) CasesC() []int {
    return []int{-4, -99, 9}
}

func (*Suite) TestAddButParametrized(t T, p struct{ A, B, C int }) {
    testo.Run(t, "commutative", func(t T) {
        if Add(p.A, p.B) != Add(p.B, p.A) {
            t.Errorf("%[1]d + %[2]d != %[2]d + %[1]d", p.A, p.B)
        }
    })

    testo.Run(t, "associative", func(t T) {
        if Add(Add(p.A, p.B), p.C) != Add(p.A, Add(p.B, p.C)) {
            t.Errorf("(%[1]d + %[2]d) + %[3]d != %[1]d + (%[2]d + %[3]d)", p.A, p.B, p.C)
        }
    })
}
```

## Plugins

One of the main features of Testo is the plugin system.

### Writing Plugins

Plugin examples:

1. A plugin that runs tests in reverse order.
2. A plugin that extends the `t.Log` method.
3. A plugin that adds new methods for `T` (fixtures).
4. A plugin that measures the time of each test.

```go
import (
    "github.com/ozontech/testo"
    "github.com/ozontech/testo/plugin"
)

// We can embed testo.T in plugins - it will point to the same T
// as in the current test.
type ReverseTestsOrder struct{ *testo.T }

// Plugins must implement this function
// so that testo understands it's a plugin and can handle it correctly.
func (*ReverseTestsOrder) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
    return testoplugin.Spec{
        Plan: testoplugin.Plan{
            Modify: func(tests *[]testoplugin.PlannedTest) {
                slices.Reverse(*tests)
            },
        },
    }
}

type OverrideLog struct { *testo.T }

func (*OverrideLog) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
    return testoplugin.Spec{
        Overrides: testoplugin.Overrides{
            Log: func(f testoplugin.FuncLog) testoplugin.FuncLog {
                return func(args ...any) {
                    // will be called each time t.Log is called
                    fmt.Println("Inside log override")
                    f(args...)
                }
            },
        },
    }
}

type AddNewMethods struct{ *testo.T }

func (*AddNewMethods) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
    // We have nothing to use here, so we return an empty specification
    return testoplugin.Spec{}
}

// Later we will see how we can call this method from tests.
func (a *AddNewMethods) Explode() { a.Fatal("BOOM") }

type Timer struct {
    *testo.T
    start time.Time
}

func (t *Timer) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
    return testoplugin.Spec{
        Hooks: testoplugin.Hooks{
            BeforeEach: testoplugin.Hook{
                Priority: testoplugin.TryLast,
                Func: func() {
                    // the .Plugin method is called for each (sub-)test,
                    // so we can modify fields without risk of synchronization errors.
                    t.start = time.Now()
                },
            },
            AfterEach: testoplugin.Hook{
                Priority: testoplugin.TryFirst,
                Func: func() {
                    elapsed := time.Since(t.start)

                    fmt.Printf("Test %q took %s\n", t.Name(), elapsed)
                },
            },
        },
    }
}
```

### Using Plugins

Recall the alias for `T` defined earlier:

```go
type T = *testo.T
```

Now we can add (install) plugins to it:

```go
type T struct{
    *testo.T
    *ReverseTestsOrder
    *OverrideLog
    *AddNewMethods
    *Timer
}
```

That's the only change.
Testo will take care of the rest.

And since `AddNewMethods` is now embedded in `T`, we can use its methods without magic:

```go
func (*Suite) TestBoom(t T) {
    t.Explode()
}
```

> [!NOTE]
> Plugins must be installed as pointers.
>
> Pointers allow plugins to share their state with other plugins,
> by pointing to the same memory location through pointers.

## Running tests without suites

It's possible:

```go
type T struct{
    *testo.T
    *ReverseTestsOrder
    *OverrideLog
    *AddNewMethods
    *Timer
}

func TestFoo(t *testing.T) {
    testo.RunSuite(t, func(t T) {
        t.Log("Hello from testo!")
    })
}
```

Or, if you need to run several tests from a single "real" test:

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
