# Examples

Showcase of various Testo features.

Examples are sorted by their simplicity in ascending order from basic to advanced.
You may want to start from the simplest
examples such as [01_suite](./01_suite/main_test.go) or [01_suiteless](./01_suiteless/main_test.go).

To run each test execute the following command in the example directory:

```bash
go test . -v -tags example -count=1
```

Or, if you can run `make` to do the same.

Each example includes its output in `output.golden` file.
