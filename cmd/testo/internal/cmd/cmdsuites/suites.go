package cmdsuites

import (
	"flag"
	"fmt"
	"go/token"
	"path/filepath"

	"github.com/ozontech/testo/cmd/testo/internal/cli"
	"github.com/ozontech/testo/cmd/testo/internal/cmd"
	"github.com/ozontech/testo/cmd/testo/internal/loader"
)

func init() {
	cli.Add("suites", func(f *flag.FlagSet, c *Cmd) {
		c.Format.Set("{{ .Suite }}")

		f.StringVar(&c.Load.Tags, "tags", cmd.DefaultTags, "build tags separated by comma")
		f.StringVar(&c.Load.Testo, "testo", cmd.DefaultTesto, "testo package")
		f.Var(&c.Format, "f", "output format")
		f.BoolVar(&c.Nul, "0", false, "output each line delimited by NUL byte")
	},
		cli.WithShort("Show testo suites"),
		cli.WithUsage(`[flags] [pattern...] [flags]

Format:
  flag -f accepts Go text/template string with the following data as input:

	type Data struct {
		Package string
		Suite   Entity
		Test    Entity
		Tests   []Entity
	}

	type Entity struct {
		Name string
		Pos  Pos
	}

	type Pos struct {
		Dir      string
		Filename string
		Path     string
		Line     int
		Column   int
	}

Examples:
  pick suite test with fzf and bat preview

  	testo suites ./... -0 -f '{{ .Package }}.{{ .Suite.Name }}.{{ .Test.Name }} {{ .Test.Pos.Path }} {{ .Test.Pos.Line }}' | fzf --read0 --delimiter " " --with-nth 1 --preview 'bat -Ss --color always --plain --tabs 4 --line-range {3}:+$FZF_PREVIEW_LINES {2}' --accept-nth 1 --preview-window up
`),
	)
}

type Cmd struct {
	Load   loader.Config
	Format cli.FlagTemplate
	Nul    bool
}

func (c Cmd) Run(patterns ...string) error {
	suites, err := loader.Load(c.Load, patterns...)
	if err != nil {
		return err
	}

	seen := make(map[string]bool)

	for _, s := range suites {
		err := c.printSuite(s, seen)
		if err != nil {
			return err
		}
	}

	return nil
}

type Pos struct {
	Dir      string
	Filename string
	Path     string
	Line     int
	Column   int
}

func (p Pos) String() string {
	return fmt.Sprintf("%s:%d:%d", p.Path, p.Line, p.Column)
}

type Entity struct {
	Name string
	Pos  Pos
}

func (e Entity) String() string {
	return e.Name
}

func (c Cmd) printSuite(suite loader.Suite, seen map[string]bool) error {
	newPos := func(p token.Position) Pos {
		return Pos{
			Dir:      filepath.Dir(p.Filename),
			Filename: filepath.Base(p.Filename),
			Path:     p.Filename,
			Line:     p.Line,
			Column:   p.Column,
		}
	}

	type Data struct {
		Package string
		Suite   Entity
		Test    Entity
		Tests   []Entity
	}

	data := Data{
		Package: suite.Package.Name,
		Suite: Entity{
			Name: suite.Name,
			Pos:  newPos(suite.FSet.Position(suite.Pos)),
		},
	}

	for _, t := range suite.Tests {
		data.Tests = append(data.Tests, Entity{
			Name: t.Name,
			Pos:  newPos(suite.FSet.Position(t.Pos)),
		})
	}

	for _, t := range suite.Tests {
		data.Test = Entity{
			Name: t.Name,
			Pos:  newPos(suite.FSet.Position(t.Pos)),
		}

		line, err := c.Format.Execute(data)
		if err != nil {
			return err
		}

		if seen[line] {
			continue
		}

		seen[line] = true

		if c.Nul {
			fmt.Print(line, "\x00")
		} else {
			fmt.Println(line)
		}
	}

	return nil
}
