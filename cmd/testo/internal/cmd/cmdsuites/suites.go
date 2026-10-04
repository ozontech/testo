package cmdsuites

import (
	"errors"
	"flag"
	"fmt"
	"go/token"
	"os"
	"path/filepath"

	"github.com/ozontech/testo/cmd/testo/internal/cli"
	"github.com/ozontech/testo/cmd/testo/internal/cmd"
	"github.com/ozontech/testo/cmd/testo/internal/loader"
)

func init() {
	cli.Add(
		"suites", func(f *flag.FlagSet, c *Cmd) {
			// the default template is a valid constant
			_ = c.Format.Set("{{ .Package }}/{{ .Suite }}")

			f.StringVar(
				&c.Load.Tags,
				"tags",
				"",
				"build tags separated by comma, derived from source if empty",
			)
			f.StringVar(&c.Load.Testo, "testo", cmd.DefaultTesto, "testo package")
			f.Var(&c.Format, "f", "output format")
			f.BoolVar(&c.Nul, "0", false, "output each line delimited by NUL byte")
			f.BoolVar(&c.Deduplicate, "dedup", false, "deduplicate output line")
		},
		cli.WithShort("Show testo suites"),
		cli.WithUsage(`[flags] [pattern...] [flags]

Format:
  flag -f accepts Go text/template string with the following data as input:

`+templateTypes("\t")+`

Examples:
  pick suite test with fzf and bat preview

  	testo suites ./... -0 `+
			`-f '{{ .Package }}/{{ .Suite }}.{{ .Test }} {{ .Test.Pos.Path }} {{ .Test.Pos.Line }}' | `+
			`fzf --read0 --delimiter " " --with-nth 1 `+
			`--preview 'bat -Ss --color always --plain --tabs 4 `+
			`--line-range {3}:+$FZF_PREVIEW_LINES {2}' --accept-nth 1 --preview-window up

  output as json and filter with jq

  	testo suites ./... -f '{{ json .Test }}' | jq '. | select(.Parametrized)'
`),
	)
}

type Cmd struct {
	Load        loader.Config
	Format      cli.FlagTemplate
	Nul         bool
	Deduplicate bool
}

func (c Cmd) Run(patterns ...string) error {
	suites, err := loader.Load(c.Load, patterns...)
	if err != nil {
		var errLoad *loader.LoadError

		if !errors.As(err, &errLoad) {
			return err
		}

		// diagnostics are warnings here: show the suites that did load,
		// reporting problems is the job of "testo lint"
		for _, d := range errLoad.Diagnostics {
			d.Println(os.Stderr, errLoad.FSet)
		}
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

func newPos(p token.Position) Pos {
	return Pos{
		Dir:      filepath.Dir(p.Filename),
		Filename: filepath.Base(p.Filename),
		Path:     p.Filename,
		Line:     p.Line,
		Column:   p.Column,
	}
}

func newParams(ps []loader.Parameter) []Parameter {
	s := make([]Parameter, 0, len(ps))

	for _, p := range ps {
		s = append(s, Parameter{Name: p.Name})
	}

	return s
}

func newTest(fset *token.FileSet, t loader.SuiteTest) Test {
	return Test{
		Name:         t.Name,
		Pos:          newPos(fset.Position(t.Pos)),
		Parametrized: t.Parametrized,
		Parameters:   newParams(t.Parameters),
	}
}

func (c Cmd) printSuite(suite loader.Suite, seen map[string]bool) error {
	data := Data{
		Package: Package{
			Name: suite.Package.Name,
			Path: suite.Package.Path,
			Dir:  suite.Package.Dir,
		},
		Suite: Suite{
			Name: suite.Name,
			Pos:  newPos(suite.FSet.Position(suite.Pos)),
		},
	}

	for _, t := range suite.Tests {
		data.Tests = append(data.Tests, newTest(suite.FSet, t))
	}

	emit := func(t Test) error {
		data.Test = t

		line, err := c.Format.Execute(data)
		if err != nil {
			return err
		}

		if c.Deduplicate && seen[line] {
			return nil
		}

		seen[line] = true

		if c.Nul {
			fmt.Print(line, "\x00")
		} else {
			fmt.Println(line)
		}

		return nil
	}

	// a suite without tests is still worth showing
	if len(suite.Tests) == 0 {
		return emit(Test{})
	}

	for _, t := range suite.Tests {
		if err := emit(newTest(suite.FSet, t)); err != nil {
			return err
		}
	}

	return nil
}
