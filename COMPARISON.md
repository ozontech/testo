# Comparison with other frameworks

> [!NOTE]
> [Testify] is mostly known as an assertion library, not a framework.
> Testo isn't an alternative to testify's `require` & `assert` and can be used with or without it.

|                      | [Testo]           | [Ginkgo]                 | [GoConvey] | [Godog]                 | [Testify] | [Allure-Go] |
| :------------------- | :---------------: | :----------------------: | :--------: | :---------------------: | :-------: | :---------: |
| Plugins              | Yes               | No                       | No         | No                      | No        | No          |
| Native `go test`[^7] | Yes               | No                       | No         | Yes                     | Yes       | Yes         |
| Suites & hooks       | Yes               | Yes                      | Yes        | Yes                     | Yes       | Yes         |
| Parametrized tests   | Yes               | Yes                      | No         | Yes                     | No        | Yes         |
| Parallel tests       | Yes               | Partially[^1]            | No[^2]     | Yes                     | No[^3]    | Yes         |
| DSL[^4]              | No                | Yes (BDD)                | Yes (BDD)  | Yes (BDD with Cucumber) | No        | No          |
| Stdlib only[^5]      | Yes               | No                       | No         | No                      | No        | No          |
| Reporting            | Via plugins[^6]   | JUnit, Teamcity & Custom | HTML       | JUnit & Cucumber        | None      | Allure      |

Testo focuses on extensibility through plugins and stays a thin layer over usual tests.
Other frameworks may be a better fit if you need BDD scenarios or some unique features "out of the box".
Testo can support BDD-style tests through plugins.

[^1]: Ginkgo runs parallel tests in [separate processes](https://onsi.github.io/ginkgo/#mental-model-how-ginkgo-runs-parallel-specs) with its own runner (not available through `go test`). This is _less performant_ than native `go test` parallelization based on goroutines.
[^2]: [Not supported](https://github.com/smartystreets/goconvey/issues/360).
[^3]: See [issue #934](https://github.com/stretchr/testify/issues/934).
[^4]: DSL — Domain-Specific Language. Requires describing tests in a specific way, different from usual Go tests. Not necessarily a bad thing, but has a learning curve and is less flexible.
[^5]: Whether it has any dependencies. Not necessarily a bad thing, but fewer dependencies mean a smaller footprint, faster build times and avoiding potential vulnerabilities.
[^6]: Any report format is achievable through plugins, but none is baked into Testo by default. See [testo-allure](https://github.com/ozontech/testo-allure).
[^7]: "Native `go test`" means _all features_ are supported using only the `go test` command without any other CLIs.

[Testo]: https://github.com/ozontech/testo
[Ginkgo]: https://github.com/onsi/ginkgo
[Testify]: https://github.com/stretchr/testify
[GoConvey]: https://github.com/smartystreets/goconvey
[Godog]: https://github.com/cucumber/godog
[Allure-Go]: https://github.com/ozontech/allure-go
