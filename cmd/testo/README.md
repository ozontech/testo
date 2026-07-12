# Testo CTL

Command line tool for Testo.

> [!WARNING]
> This tool is experimental, handle with care;
> may change without warning

## Features

- Linter for common mistakes with Testo (wrong `T` type, mismatched params and more)
- Suites runner
- Suites explorer

## Install

```bash
go install github.com/ozontech/testo/cmd/testo
```

## Usage

Run `testo -h` to see available commands:

```txt
Usage:
  testo [command]

Available Commands:
  lint       Run testo linter
  run        Run testo suites
  suites     Show testo suites
  tags       Show project build tags
  version    Show testo version
```

Run `testo [command] -h` to show help for the given command.
