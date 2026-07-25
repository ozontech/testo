# Examples

Small runnable projects, one per Testo feature.

| Example | What it shows |
| :------ | :------------ |
| [01_suite](./01_suite/main_test.go) | A minimal suite with a few tests. |
| [01_suiteless](./01_suiteless/main_test.go) | Running tests without a suite. |
| [02_hooks](./02_hooks/main_test.go) | Suite hooks: `BeforeAll`, `BeforeEach`, `AfterEach`, `AfterAll`. |
| [03_parametrized](./03_parametrized/main_test.go) | Parametrized tests with `CasesXxx` methods. |
| [04_plugins](./04_plugins/main_test.go) | Writing plugins: hooks, method overrides, test planning, new `T` methods. |
| [05_parallel](./05_parallel/main_test.go) | Parallel tests and how hooks behave around them. |
| [06_errors](./06_errors/main_test.go) | Error messages for common mistakes. **Fails on purpose.** |
| [07_annotations](./07_annotations/main_test.go) | Attaching static options to tests with `testo.For` & `testo.ForEach`. |
| [08_subsuites](./08_subsuites/main_test.go) | Nesting suites with `testo.RunSubSuite`. |

To run an example, execute the following command in its directory
(or run `make`, which does the same):

```bash
go test . -v -tags example -count=1
```

Each example includes its expected output in an `output.golden` file.
