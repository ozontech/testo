# Testo CTL

Auxiliary command line tool for Testo featuring linter, suites explorer and runner.

> [!WARNING]
> This tool is experimental, handle with care;
> may change without warning

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
