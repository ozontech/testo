package cmdversion

import (
	"flag"
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/ozontech/testo/cmd/testo/internal/cli"
)

func init() {
	cli.Add(
		"version",
		func(*flag.FlagSet, *Cmd) {},
		cli.WithShort("Show testo cli version"),
		cli.WithoutArgs(),
	)
}

type Cmd struct{}

func (Cmd) Run(...string) error {
	version := "unknown"

	info, ok := debug.ReadBuildInfo()
	if ok {
		version = info.Main.Version
	}

	// NOTE: this command only shows testo-cli (this binary)
	// version and not testo framework version from go.mod of a project
	// because parsing go.mod requires golang.org/x/mod/modfile
	// and we strictly avoid any dependencies.

	fmt.Printf("testo version %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)

	return nil
}
