package cmdlint

import (
	"bytes"
	"errors"
	"flag"

	"github.com/ozontech/testo/cmd/testo/internal/cli"
	"github.com/ozontech/testo/cmd/testo/internal/cmd"
	"github.com/ozontech/testo/cmd/testo/internal/loader"
)

func init() {
	cli.Add("lint", func(f *flag.FlagSet, c *Cmd) {
		f.StringVar(&c.Load.Tags, "tags", cmd.DefaultTags, "build tags separated by comma")
		f.StringVar(&c.Load.Testo, "testo", cmd.DefaultTesto, "testo package")
		f.BoolVar(&c.Load.Strict, "strict", false, "enable strict mode")
		f.BoolVar(&c.JSON, "json", false, "output json")
	}, cli.WithShort("Run testo linter"))
}

type Cmd struct {
	Load loader.Config
	JSON bool
}

func (c Cmd) Run(patterns ...string) error {
	_, err := loader.Load(c.Load, patterns...)
	if err == nil {
		return nil
	}

	var errLoad *loader.LoadError

	if !errors.As(err, &errLoad) {
		return err
	}

	var buf bytes.Buffer

	for _, d := range errLoad.Diagnostics {
		if c.JSON {
			d.JSON(&buf, errLoad.FSet)
		} else {
			d.Print(&buf, errLoad.FSet)
		}
	}

	return cli.Exit(1).Stdout(buf.String())
}
