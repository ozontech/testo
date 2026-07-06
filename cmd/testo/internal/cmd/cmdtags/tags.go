package cmdtags

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ozontech/testo/cmd/testo/internal/cli"
	"github.com/ozontech/testo/cmd/testo/internal/loader"
)

func init() {
	cli.Add("tags", func(f *flag.FlagSet, cmd *Cmd) {
		f.BoolVar(&cmd.Tests, "tests", false, "only show build tags used in *_test.go files")
		f.BoolVar(&cmd.All, "a", false, "include build tags cancelled by negations, e.g. //go:build !tag")
	},
		cli.WithShort("Show project build tags"),
		cli.WithLong(`Show project build tags.

It traverses all go files and parses //go:build directives.
Use -tests flag to traverse only *_test.go files.

If same tag is both required and cancelled by different expressions it will be omitted.
Pass -a flag to change that.

	//go:build mytag
	//go:build !mytag

This command must be executed from the same directory as go module (project).
`),
		cli.WithoutArgs(),
	)
}

type Cmd struct {
	All   bool
	Tests bool
}

func (c Cmd) Run(...string) error {
	add, remove, err := loader.BuildTags(context.Background(), c.Tests)
	if err != nil {
		return err
	}

	if !c.All {
		for k := range remove {
			delete(add, k)
		}
	}

	if len(add) > 0 {
		fmt.Println(join(keys(add)))
	}

	return nil
}

func join(s []string) string {
	return strings.Join(s, ",")
}

func keys[M ~map[K]V, K cmp.Ordered, V any](m M) []K {
	s := slices.Collect(maps.Keys(m))

	slices.Sort(s)

	return s
}
