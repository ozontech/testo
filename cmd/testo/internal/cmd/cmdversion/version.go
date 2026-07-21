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
		cli.WithShort("Show testo version"),
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

	fmt.Printf("testo version %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)

	return nil
}
