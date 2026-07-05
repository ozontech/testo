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
	},
		cli.WithShort("Show project build tags"),
		cli.WithoutArgs(),
	)
}

type Cmd struct {
	Tests bool
}

func (c Cmd) Run(...string) error {
	add, remove, err := loader.BuildTags(context.Background(), c.Tests)
	if err != nil {
		return err
	}

	conflicting := intersection(add, remove)

	if len(conflicting) > 0 {
		return cli.Exit(2).Stderr("conflicting tags: " + join(keys(conflicting)))
	}

	for r := range remove {
		delete(add, r)
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

func intersection[M ~map[K]struct{}, K comparable](a, b M) M {
	m := make(M)

	for k := range a {
		if _, ok := b[k]; ok {
			m[k] = struct{}{}
		}
	}

	for k := range b {
		if _, ok := a[k]; ok {
			m[k] = struct{}{}
		}
	}

	return m
}
