# Comparison with other frameworks

> [!NOTE]
> [Testify] is mostly known as an assertion library, not a framework.
> Testo isn't a testify's `require` & `assert` alternative and can be used with or without it.

|                        | [Testo]        | [Ginkgo]      | [GoConvey] | [Godog]    | [Testify]'s suites |
| :--------------------- | :------------: | :-----------: | :--------: | :--------: | :----------------: |
| Plugins                | Yes            | No            | No         | No         | No                 |
| Suites & hooks         | Yes            | Yes           | Yes        | Yes        | Yes                |
| Parametrized tests     | Yes            | Yes           | No         | Yes        | No                 |
| Parallel tests         | Yes            | Partially[^1] | No[^2]     | Yes        | No[^3]             |
| DSL[^4]                | No             | Yes (BDD)     | Yes (BDD)  | Yes (BDD)  | No                 |
| External deps[^5]      | 0              | 28            | 5          | 15         | 2                  |
| Reporting              | Via plugins[^6]| Yes           | Yes        | Yes        | No                 |

Testo focuses on extensibility through plugins and stays a thin layer over usual tests.
Other frameworks may be a better fit if you need BDD scenarios or some unique features "out of the box".
Testo can also support BDD-style tests through plugins.

[^1]: Ginkgo runs parallel tests in [separate processes](https://onsi.github.io/ginkgo/#mental-model-how-ginkgo-runs-parallel-specs) with its own runner (not available through `go test`). This is _less performant_ than native `go test` parallelization based on goroutines.
[^2]: [Not supported](https://github.com/smartystreets/goconvey/issues/360).
[^3]: See [issue #934](https://github.com/stretchr/testify/issues/934).
[^4]: DSL — Domain-Specific Language. Requires describing tests in a specific way, different from usual Go tests. Not necessarily a bad thing, but has a learning curve and is less flexible.
[^5]: Total external dependencies in `go.mod` as of July 2026. Not necessarily a bad thing, but fewer dependencies mean a smaller footprint, avoiding potential vulnerabilities and slower build times.
[^6]: Any report format is achievable through plugins. See [testo-allure](https://github.com/ozontech/testo-allure).

[Testo]: https://github.com/ozontech/testo
[Ginkgo]: https://github.com/onsi/ginkgo
[Testify]: https://github.com/stretchr/testify
[GoConvey]: https://github.com/smartystreets/goconvey
[Godog]: https://github.com/cucumber/godog
