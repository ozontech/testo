# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

## Added

- Auxiliary command line tool for Testo featuring linter, suites explorer and runner.

## [1.5.1] - 2026-06-17

### Fixed

- Fixed a bug when skipped tests in suite would trigger "no suite tests" warning.

## [1.5.0] - 2026-06-14

### Added

- Added strict mode which turns warnings into hard errors. Can be enabled with `-testo.strict` command line argument and `TESTO_STRICT` environment variable (lower priority).

### Changed

- Testo error messages are now more descriptive.
- Malformed method names (e.g. lowercase letter after `Test` prefix) raise fatal errors, similar to native `go test` runner.

## [1.4.0] - 2026-06-07

### Added

- Support for setting plugin priority (order) for `Plan` and `Overrides` components.

### Fixed

- Add missing call to `t.Helper` in runner.

## [1.3.2] - 2026-05-30

### Fixed

- Fixed a data-race when running sub-tests in separate goroutines.

## [1.3.1] - 2026-05-29

### Added

- VS Code extension snippets for common Testo blocks.

### Fixed

- Fixed a bug when long cache keys could trigger an error.

## [1.3.0] - 2026-05-24

### Added

- Ability to run tests without suites.

### Changed

- `testo.Options` now returns an empty struct to enable `var _ = testo.Options(...)` usage.

## [1.2.0] - 2026-05-21

### Added

- Ability to run test sub-suites.
- Provide FuncPC for sub-tests.

## [1.1.0] - 2026-05-20

### Added

- Support for overriding `t.Chdir` and `t.Cleanup`.
- Access to `testing.T` which invoked a suite through reflection.

### Changed

- Improve log format about collected plugins. Do not show it if zero plugins are collected.
- `t.Setenv` now uses defined overrides for cleanup and failures.

## [1.0.1] - 2026-05-15

### Fixed

- Reverse order of level options so that child options come after parent options.

## [1.0.0] - 2026-05-13

### Added

- Initial stable version.

[1.5.1]: https://github.com/ozontech/testo/compare/v1.5.0...v1.5.1
[1.5.0]: https://github.com/ozontech/testo/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/ozontech/testo/compare/v1.3.2...v1.4.0
[1.3.2]: https://github.com/ozontech/testo/compare/v1.3.1...v1.3.2
[1.3.1]: https://github.com/ozontech/testo/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/ozontech/testo/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/ozontech/testo/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/ozontech/testo/compare/v1.0.1...v1.1.0
[1.0.1]: https://github.com/ozontech/testo/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/ozontech/testo/releases/tag/v1.0.0
