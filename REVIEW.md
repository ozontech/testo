# cmd/testo review

Code review of the auxiliary CLI (`cmd/testo`): bugs found, how they were
reproduced, and improvement vectors. All bugs listed below are fixed.

## Confirmed bugs

### 1. `testo run` could silently run zero tests (false green)

`loadRunners` (`internal/loader/runner.go`) tracked the enclosing test function
by remembering the last `FuncDecl` whose name passes `parse.IsTest` — without
checking `Recv == nil` and without resetting on non-test functions. Suite
methods like `func (Suite) TestFoo(t T)` are `FuncDecl`s too, so a `RunSuite`
call inside a plain helper was attributed to the previous method or test
declared in the file.

Reproduced: with a helper function calling `RunSuite`, the tool generated
`go test -run '^TestFoo$/^Suite$'` (the *suite method* name); `go test`
reported "no tests to run, PASS" — exit 0.

Fix: only function declarations without a receiver update the current test
name, and it is reset on non-test functions. A suite whose `RunSuite` caller
cannot be found now produces a warning instead of a silently wrong `-run`
pattern.

### 2. `-- [test flags]` was broken without a pattern

`parseFlagSet` (`internal/cli/cli.go`) collected positionals and re-parsed
them, losing the `--` separator. `testo run -n -- -count=1` died with
`flag provided but not defined: -count`.

Fix: `--` is detected and everything after it is kept as-is; the final
re-parse is performed behind a synthetic `--` terminator so flag-like
positionals are never parsed as the command's own flags.

### 3. One bad package made `suites`/`run` unusable for the whole repo

`loader.Load` deliberately returns `(suites, *LoadError)`, but `cmdsuites`
and `cmdrun` treated any error as fatal. Reproduced on this repository:
`testo suites ./...` printed nothing because `examples/06_errors`
(intentionally malformed) produces diagnostics.

Fix: both commands report diagnostics as warnings on stderr and proceed with
the suites that did load. Only `lint` treats diagnostics as failure.

### 4. Suites were collected twice via `go list -test` package variants

`packageslite` skipped only the `pkg.test` binary; both `pkg` and
`pkg [pkg.test]` were type-checked and scanned, so a suite declared in a
non-test file appeared twice. The default `suites` format hid it via output
dedup, but `-f '{{ .Package.Path }}/...'` printed both, and `run` did the
runner scan twice.

Fix: when the test variant of a package is present the base package is
dropped (the variant is a strict superset of its files), and the variant's
import path is normalized (the ` [pkg.test]` suffix stripped) after type
checking. Runner matching compares suites by normalized package path + type
name rather than type identity, so suites and their callers agree across
variants.

### 5. Diagnostics printed out of source order

`Load` sorted diagnostics with unstable `slices.SortFunc` comparing file
names only. Reproduced: lines 23, 28, 35, **16**, 47 reported in that order
for a single file.

Fix: diagnostics sort by (file name, position).

## Minor bugs

- **Dead/inconsistent code in `asSuite`** (`internal/loader/loader.go`):
  `if invalidParams { continue }` sat *after* the append as the last statement
  of the loop body — a no-op. A 1-parameter test with the wrong `T` was still
  listed while a 2-parameter test with the wrong `T` was skipped. Fixed:
  a test with any signature diagnostic is not listed.
- **Auto-derived tags included implicit tags.** `BuildTags` collects every
  tag in the module, so a `//go:build windows` file added `windows` to
  `-tags`, pulling mutually exclusive platform files into type checking.
  `Load` also silently swallowed the `BuildTags` error. Fixed: implicit tags
  (GOOS/GOARCH values, `go1.*`, `cgo`, `race`, …) are excluded from
  derivation (but still shown by `testo tags`), and the error propagates.
- **Panic risk in `asT`** (`internal/loader/loader.go`): four chained
  unchecked type assertions on the internal layout of `testo.Suite[T]`
  (`struct{ _ [0]*T }`). Version skew between the installed CLI and the
  project's testo turned into a panic instead of a non-match. Fixed with
  checked assertions.
- **`goList` decoded `Error` but never checked it** (uses `go list -e`), so
  broken packages surfaced later as an opaque `package %q is not complete`.
  Fixed: a directly-matched package with files and a list error fails with
  that error; matched directories excluded by build constraints are skipped.
- **Empty `-tags ''` argument** was passed to `go test` when no tags were
  derived, and empty strings survived tag splitting. Fixed.

## Improvement vectors (applied)

- Usage text printed raw `os.Args[0]` — under `go run` that is the full
  go-build cache path. Now uses `filepath.Base`.
- `loadRunners` cached runners for every suite type *except* the queried
  one, and returned map keys in nondeterministic order (also affecting the
  package argument order of the built `go test` command, i.e. `-n` output).
  Now scans once, caches everything, and output is sorted.
- `testo suites` printed nothing for a suite with zero tests. Now the suite
  line is printed once (with an empty `.Test`).
- Added regression tests for CLI flag parsing (`internal/cli`).

## Improvement vectors (not applied)

- Golden end-to-end tests for `lint`/`suites`/`run -n` output over
  `examples/` would lock in exit codes and diagnostic ordering cheaply,
  matching the framework's own golden-file approach (`examples_test.go`).
- The `suite.test` regex pattern of `testo run` splits on the *first* `.`,
  so a suite regex containing `.` (e.g. `Test.+` as a suite-only pattern)
  cannot be expressed.
