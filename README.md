[![Testo banner showing its chef gopher mascot on a blue background](./banner.svg)](https://github.com/ozontech/testo)

# Testo

[![Go Reference](https://pkg.go.dev/badge/github.com/ozontech/testo.svg)](https://pkg.go.dev/github.com/ozontech/testo)
[![Go Report Card](https://goreportcard.com/badge/github.com/ozontech/testo)](https://goreportcard.com/report/github.com/ozontech/testo)
[![Code Coverage](https://github.com/ozontech/testo/raw/gh-pages/coverage.svg?raw=true)](https://ozontech.github.io/testo/coverage.html)
[![Quality Assurance](https://github.com/ozontech/testo/actions/workflows/qa.yml/badge.svg)](https://github.com/ozontech/testo/actions/workflows/qa.yml)

Testo is a modular testing framework for Go built on top of `testing.T`
with an extensive plugin system.

> Testo (/tɛstɒ/) is a play on words "test" and "тесто", meaning "dough".
> Just like you can cook anything from dough, you can test anything with Testo!

Add some flavor to your tests with
[toppings - a collection of small, miscellaneous plugins for Testo framework.](https://github.com/ozontech/testo-toppings)

## Features

- [Plugins](./examples/04_plugins/main_test.go) - adapt your tests to any scenario with features you need.
- [Parametrized tests](./examples/03_parametrized/main_test.go) - describe a test once, repeat it with different parameters.
- [Parallel tests](./examples/05_parallel/main_test.go) - make your tests faster by running them all at once.
- [Lifecycle hooks](./examples/02_hooks/main_test.go) - before and after any suite, test & sub-test.
- [Test annotations](./examples/07_annotations/main_test.go) - attach static options to any test.
- [Informative errors and traces](./examples/06_errors/main_test.go) - no need to guess what went wrong.
- Sub-tests & sub-suites - support for nested tests and nested suites.
- Test reflection - deeply inspect test's meta-information.
- Caching - key-value storage persistent between test runs.

## Why Testo

At Ozon, Testo powers thousands of end-to-end tests daily in production.

With plugins, it is flexible enough to adapt to diverse requirements,
without leaving the Go ecosystem - just a layer over `testing.T`.

If your needs are outgrowing standard `testing` package, Testo is a great choice.

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

But there is more!
Testo supports suites, parametrized tests & plugins, see [Next steps](#next-steps).

See also [VS Code extension for Testo](#vs-code-extension).

### Next steps

- Take [a guided tour of Testo](./docs/tutorial.md) by making simple plugins and running the tests using various features.
- See [test examples](./examples).
- Learn [how to use various Testo features](./docs/how-to.md).
- Read a [brief description and technical overview](./docs/technical-overview.md) of Testo.
- View [API documentation](https://pkg.go.dev/github.com/ozontech/testo).

## Plugins

Testo features a powerful plugin system.

Plugins can:

- Provide `BeforeAll`/`AfterAll`, `BeforeEach`/`AfterEach` & `BeforeSubEach`/`AfterSubEach` hooks.
- Plan tests for execution - filter, duplicate & reorder.
- Override built-in `T` methods, such as `Log`, `Error` and _etc._
- Extend `T` by adding new methods.
- Allow users to configure their behavior through options.
- Communicate with other plugins.
- Add command line flags for `go test` command.

Examples:

- [Testo Allure Plugin](https://github.com/ozontech/testo-allure) - enhance your tests with automatically generated [Allure Reports](https://allurereport.org/).
- [Testo Rerun Plugin](https://github.com/ozontech/testo-toppings/tree/main/rerun) - adds `--last-failed`-like behaviour from Pytest to Testo. Makes it possible to rerun only failed tests.
- [Testo XFail Plugin](https://github.com/ozontech/testo-toppings/tree/main/xfail) - adds `t.XFail()` method to mark a test as "expected to fail".
- [Testo Parallel Plugin](https://github.com/ozontech/testo-toppings/tree/main/parallel) - marks all tests as parallel by default.

## VS Code Extension

Testo has its own [VS Code extension](./vscode-extension).

Makes it easier to run and debug individual suite tests.

![VSCode extension screenshot showing codelens buttons for running and debugging a test](./vscode-extension/example.png)

## Minimum supported Go version

Testo guarantees to support at least **3 latest major** [Go releases](https://go.dev/doc/devel/release).

Currently, minimum supported Go version is **1.24**

## Contributing

Contributions are welcome!

See [contributing guidelines](./CONTRIBUTING.md).

## License

This project is released under the [Apache-2.0 license](./LICENSE).

---

> [!TIP]
> If you find Testo useful, [consider giving this repository a star](https://github.com/ozontech/testo) to help it reach more people.
> Thank you!
