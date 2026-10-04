# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Testo is a modular testing framework for Go built on top of `testing.T`, with suites, parametrized tests, lifecycle hooks, and an extensive plugin system. It is a zero-dependency library (module `github.com/ozontech/testo`, Go 1.24+).

## Commands

```bash
make                  # full check: generate, fmt, lint, test, test-examples, tidy
make test             # go test -race -shuffle=on ./...
make test-examples    # verify examples output matches golden files (go test -tags e2e ./examples_test.go)
make fmt              # golangci-lint fmt
make lint             # golangci-lint run --tests=false
make coverage         # test coverage report
make install          # install the ./cmd/testo CLI

# Run a single test
go test -race -run 'TestName' .

# Regenerate golden output for examples (after changing examples or output format)
make update-examples-output
```

Examples under `examples/` have expected output stored in `examples/*/output.txt`; `examples_test.go` (build tag `e2e`) runs each example package and diffs against those files. If you change log/error formatting or the examples, run `make update-examples-output`.

## Architecture

### Package layout

- **Root package `testo`** — the framework core. Key files:
  - `t.go` — `T` (the `testing.T` wrapper), `TestingT`, `CommonT` interfaces. Users embed `*testo.T` plus plugin pointers into their own `T` struct.
  - `construct.go` — `construct[T]`: reflective construction of user `T` types, filling embedded plugin fields. This is the core DI mechanism of the framework; plugin instances are deduplicated so all pointers to the same plugin type share one instance.
  - `runner.go` — `RunSuite`, `RunTest`, `Test`: entry points. A hidden wrapper test named `testo!` (contains `!` so it can't collide with Go identifiers) wraps suite tests so hooks work with parallel tests.
  - `collector.go` — reflection over suite types: collects `TestXxx` methods and `CasesXxx` parametrization providers, validates naming.
  - `suite.go` — `Suite[T]` base type and the suite hook interface (`BeforeAll`/`BeforeEach`/`AfterEach`/`AfterAll`).
  - `plugin.go` — merging of multiple plugin `Spec`s (plans sorted by priority, hooks/overrides chained).
  - `annotations.go` — `For`/`ForEach`: attach static options to tests via a global registry keyed by function identity.
  - `options.go`, `flag.go` — plugin options and `-testo.*` command-line flags (e.g. `-testo.strict`, env `TESTO_STRICT`).
- **`testoplugin`** — public plugin API: `Spec` = `Plan` (filter/reorder/duplicate tests before run) + `Hooks` (BeforeAll/Each/SubEach etc.) + `Overrides` (replace built-in `T` methods like `Log`, `Error`). Plugins implement a `Plugin(parent, options)` method; innermost (deepest embedded) plugins are initialized first.
- **`testoreflect`** — public read-only reflection API (`TestInfo`, `SuiteInfo`) given to plugins.
- **`testocache`** — key-value cache persistent between test runs, with namespaces.
- **`internal/`** — helpers: `parse` (method-name parsing), `testnamer`, `reflectutil`, `pragma` (`DoNotImplement` sealing), `stack`, `syncutil`, `env`.
- **`cmd/testo`** — optional auxiliary CLI (linter, suites explorer, runner: `testo lint`, `testo run`, `testo suites`, `testo tags`). Experimental.
- **`vscode-extension/`** — VS Code extension (npm/vsce, own Makefile).

### Lifecycle (see docs/technical-overview.md for full detail)

`RunSuite` runs a root test named after the suite: collects and validates tests → initializes plugins (via `.Plugin(parent, options)`) → `BeforeAll` (plugin hooks, then suite hook) → collects parametrized cases from `CasesXxx` → applies plugin plans → runs each test under the `testo!` wrapper (per-test plugin construction, `BeforeEach`/`AfterEach`; sub-tests via `testo.Run` get `BeforeEachSub`/`AfterEachSub`) → `AfterAll`. Panics in tests and per-test hooks are caught; panics in `BeforeAll`/`AfterAll` are not.

### Plugin DI model

Users compose their `T` as a struct embedding `*testo.T` and plugin pointers. Plugins can depend on other plugins by embedding them; Testo resolves the graph so every reference to a given plugin type points at the same instance. `construct.go` is where this happens — changes there affect the entire framework.

## Conventions

- Zero dependencies in the main module — do not add third-party imports to library code.
- Public API stability matters: interfaces are sealed with `internal/pragma.DoNotImplement` private methods; follow that pattern for new public interfaces.
- Every user-visible change must be described in `CHANGELOG.md` under the "Unreleased" section (Keep a Changelog format, SemVer).
- Lint runs on non-test code only (`--tests=false`); format with `golangci-lint fmt`, not plain gofmt.
- Per CONTRIBUTING.md, AI-assisted work must be disclosed in PRs (tool + extent).
