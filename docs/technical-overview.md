# Technical overview

A technical overview of Testo.

## Lifecycle

When you call `testo.RunSuite` the following happens:

```txt
A root test named the same as a suite is run {
    Suite tests are collected and verified.

    Plugins are collected and initialized with ".Plugin(parent, options)" method call, if implemented. Innermost plugins are initialized first.
    At this top level, parent is a typed-nil instance of the plugin's own type: the interface is non-nil, but the concrete pointer inside is nil.

    "BeforeAll" plugin hooks are called.
    "BeforeAll" suite hook is called.

    "CasesXXX" functions are called and parametrized tests are collected.
    Test plan from plugins is applied to the final test collection.

    A test named "testo!" is run {
        For each test in collection {
            Plugins are collected and initialized with ".Plugin(parent: parent, options)" method call, if implemented. Innermost plugins are initialized first.

            "BeforeEach" plugin hooks are called.
            "BeforeEach" suite hook is called.

            Actual test is run {
                For each sub-test in test (ran through "testo.Run") {
                    Plugins are collected and initialized with ".Plugin(parent: parent, options)" method call, if implemented. Innermost plugins are initialized first.

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

## Panics

Testo **will** catch panics from tests, including `BeforeEach`, `BeforeEachSub`, `AfterEachSub` & `AfterEach` hooks.
Other tests will run even if some tests are panicking.

Testo **will not** catch panics from `BeforeAll` & `AfterAll`.
Panic in these hooks will result in suite tests not running.

## Plugins

Testo uses dependency-injection-like mechanism to enable cross-plugin communication.

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
