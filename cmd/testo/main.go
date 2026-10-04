// Testo is an auxiliary command line tool for the testo framework
// featuring a linter, suites explorer and runner.
package main

import (
	"github.com/ozontech/testo/cmd/testo/internal/cli"
	_ "github.com/ozontech/testo/cmd/testo/internal/cmd/cmdlint"
	_ "github.com/ozontech/testo/cmd/testo/internal/cmd/cmdrun"
	_ "github.com/ozontech/testo/cmd/testo/internal/cmd/cmdsuites"
	_ "github.com/ozontech/testo/cmd/testo/internal/cmd/cmdtags"
	_ "github.com/ozontech/testo/cmd/testo/internal/cmd/cmdversion"
)

func main() {
	cli.Run()
}
