# Technical overview

How Testo works under the hood. Read this if you write plugins or
debug hook ordering.

## Mechanism

Testo works through runtime reflection over suite method sets and `T`
types. There is no code generation and no separate CLI.
This puts a few constraints on your types, checked before any
test runs:

- Plugins must be embedded as pointers. More generally, every
  exported field of `T` and of plugin structs must be a pointer type.
- Recursive plugin type references are detected and rejected.

Signature and `CasesXxx` mismatches are reported as test errors.
Violations of the type constraints above panic before any test runs,
aborting the test binary.

Plugins are re-instantiated for every test and sub-test. This is what
makes writing to plugin fields safe without synchronization, at the
cost of a constructor call per (sub-)test.

## Lifecycle

When you call `testo.RunSuite` the following happens:

```txt
A root test named the same as a suite is run {
    Suite tests are collected and verified.

    Plugins are initialized (innermost first; see note below).

    "BeforeAll" plugin hooks are called.
    "BeforeAll" suite hook is called.

    "CasesXxx" functions are called and parametrized tests are collected.
    Test plan from plugins is applied to the final test collection.

    A test named "testo!" is run {
        For each test in collection {
            Plugins are initialized (innermost first).

            "BeforeEach" plugin hooks are called.
            "BeforeEach" suite hook is called.

            Actual test is run {
                For each sub-test in test (run via "testo.Run") {
                    Plugins are initialized (innermost first).

                    "BeforeEachSub" plugin hooks are called.

                    Actual sub-test is run. For any sub-sub-...-test the same logic applies.

                    "AfterEachSub" plugin hooks are called.
                }
            }

            "AfterEach" suite hook is called.
            "AfterEach" plugin hooks are called.
        }
    }

    "AfterAll" suite hook is called.
    "AfterAll" plugin hooks are called.
}
```

Plugin initialization calls the `.Plugin(parent, options)` method,
if implemented, innermost plugins first: a plugin embedded by another
plugin initializes before the plugin embedding it. At the suite root,
`parent` is a typed-nil instance of the plugin's own type: the
interface is non-nil, but the concrete pointer inside is nil. Below
the root, `parent` is the plugin instance of the enclosing scope.

## Panics

Unlike plain `go test`, a panicking test does not abort the whole
binary: Testo catches panics from tests and from all hooks.
Other tests keep running even if some panic.

A panic in `BeforeAll` prevents the suite's tests from running.
A panic in `AfterAll` is caught like any other hook panic and fails
the suite.

If the suite-level `T` is skipped (for example, `t.Skip` inside
`BeforeAll`), the `AfterAll` hooks are skipped too.

See also [suite hooks in the tutorial](./tutorial.md#suite-hooks) and
[hook behavior with parallel tests](./how-to.md#how-to-write-parallel-tests).

## Plugins

Testo uses a dependency-injection-like mechanism for cross-plugin communication.

For example, assume we have plugin `X` and plugin `Y`.
Plugin `X` needs to interact with plugin `Y`. To make it possible, `X` needs to embed `Y`:

```go
type PluginX struct {
    *testo.T
    *PluginY
}

type PluginY struct {
    *testo.T
}

type T struct {
    *testo.T
    *PluginX
    *PluginY
}
```

Testo will keep track of referenced (requested) plugins and reuse the same instance across all pointers.
It means that both `T.PluginX.PluginY` and `T.PluginY` will point to the same instance of `PluginY`.
