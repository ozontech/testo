[![Testo banner showing its chef gopher mascot on a blue background](./banner.svg)](https://github.com/ozontech/testo)

# Testo

[![Go Reference](https://pkg.go.dev/badge/github.com/ozontech/testo.svg)](https://pkg.go.dev/github.com/ozontech/testo)
[![Code Coverage](https://github.com/ozontech/testo/raw/gh-pages/coverage.svg?raw=true)](https://ozontech.github.io/testo/coverage.html)
[![Quality Assurance](https://github.com/ozontech/testo/actions/workflows/qa.yml/badge.svg)](https://github.com/ozontech/testo/actions/workflows/qa.yml)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

Testo is a modular testing framework for Go built on top of `testing.T`
with an extensive plugin system.

> Testo (/tɛstɒ/) is a play on words "test" and "тесто", meaning "dough".
> Just like you can cook anything from dough, you can test anything with Testo!

See also [toppings](https://github.com/ozontech/testo-toppings) -
a collection of small optional plugins for Testo.

## Features

- [Plugins](./docs/plugins.md) - hook, filter and extend `T` without forking the framework ([example](./examples/04_plugins/main_test.go)).
- [Parametrized tests](./docs/how-to.md#how-to-write-parametrized-tests) - describe a test once, repeat it with different parameters ([example](./examples/03_parametrized/main_test.go)).
- [Parallel tests](./examples/05_parallel/main_test.go) - run independent tests concurrently.
- [Lifecycle hooks](./docs/tutorial.md#suite-hooks) - before and after any suite, test & sub-test ([example](./examples/02_hooks/main_test.go), [parallel caveat](./docs/how-to.md#hooks-and-parallel-sub-tests)).
- [Test annotations](./docs/how-to.md#how-to-annotate-tests) - attach static options to any test ([example](./examples/07_annotations/main_test.go)).
- [Test filtering](./docs/how-to.md#how-to-run-and-skip-specific-tests) - `-run`, `-testo.m`, and the one trap to know before writing CI filters.
- [Informative errors and traces](./examples/06_errors/main_test.go) - error messages name the exact method and type that caused them.
- [Sub-tests & sub-suites](./examples/08_subsuites/main_test.go) - support for nested tests and nested suites.
- [Test reflection](https://pkg.go.dev/github.com/ozontech/testo/testoreflect) - deeply inspect test's meta-information.
- [Caching](./docs/how-to.md#how-to-use-persistent-cache) - key-value storage persistent between test runs.
- [Zero dependencies](./go.mod).

## Why Testo

Ozon runs thousands of end-to-end tests on Testo every day.

Plugins let teams add what they need: reporting, retries, custom `T`
methods. Tests stay plain `go test` tests.

See [comparison with other frameworks](./COMPARISON.md).

## Quick Start

```bash
go get github.com/ozontech/testo
```

Your first test with Testo:

```go
package main

import (
    "testing"

    "github.com/ozontech/testo"
)

func Test(t *testing.T) {
    testo.RunTest(t, func(t *testo.T) {
        t.Log("Hello, Testo!")
    })
}
```

And run it with `go test` as usual:

```bash
go test .
```

The part Testo exists for is suites with parametrized tests.
In the same file:

```go
type Suite struct{ testo.Suite[*testo.T] }

func (Suite) CasesWord() []string {
    return []string{"dough", "bread"}
}

func (Suite) TestLen(t *testo.T, p struct{ Word string }) {
    if len(p.Word) == 0 {
        t.Error("word must not be empty")
    }
}

func TestSuite(t *testing.T) {
    testo.RunSuite(t, new(Suite))
}
```

`TestLen` runs once per word from `CasesWord`. Plugins install the
same way - as fields of a struct.

### Next steps

- Take [a guided tour of Testo](./docs/tutorial.md) by making simple plugins and running the tests using various features.
- See [test examples](./examples).
- Learn [how to use Testo features](./docs/how-to.md).
- Migrating from testify or allure-go? See the [migration guide](./docs/migration.md).
- Read the [technical overview](./docs/technical-overview.md) - lifecycle, panics, plugin internals.
- View [API documentation](https://pkg.go.dev/github.com/ozontech/testo).

## Plugins

Plugins can:

- Provide `BeforeAll`/`AfterAll`, `BeforeEach`/`AfterEach` & `BeforeEachSub`/`AfterEachSub` hooks.
- Plan tests for execution - filter, duplicate & reorder.
- Override built-in `T` methods, such as `Log` and `Error`.
- Extend `T` by adding new methods.
- Allow users to configure their behavior through options.
- Communicate with other plugins.
- Add command line flags for `go test` command.

See [the guide on writing plugins](./docs/plugins.md).

The plugin API (`testoplugin` package) follows the same
[SemVer](https://semver.org) compatibility promise as the rest of the module:
no breaking changes within a major version. New `Spec` fields may be added in minor releases.

Examples:

- [Testo Allure Plugin](https://github.com/ozontech/testo-allure) - enhance your tests with automatically generated [Allure Reports](https://allurereport.org/).
- [Testo Rerun Plugin](https://github.com/ozontech/testo-toppings/tree/main/rerun) - adds `--last-failed`-like behaviour from Pytest to Testo. Makes it possible to rerun only failed tests.
- [Testo XFail Plugin](https://github.com/ozontech/testo-toppings/tree/main/xfail) - adds `t.XFail()` method to mark a test as "expected to fail".
- [Testo Parallel Plugin](https://github.com/ozontech/testo-toppings/tree/main/parallel) - marks all tests as parallel by default.

## VS Code Extension

Testo has its own [VS Code extension](./vscode-extension).

The extension adds run/debug buttons for individual suite tests, plus snippets.

![VSCode extension screenshot showing codelens buttons for running and debugging a test](./vscode-extension/example.png)

## Minimum supported Go version

Testo guarantees to support at least **3 latest major** [Go releases](https://go.dev/doc/devel/release).

Currently, the minimum supported Go version is **1.24**.

## Contributing

Contributions are welcome!

See [contributing guidelines](./CONTRIBUTING.md).

## License

This project is released under the [Apache-2.0 license](./LICENSE).
