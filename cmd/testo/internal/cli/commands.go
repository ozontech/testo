package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/ozontech/testo/cmd/testo/internal/loader"
)

func init() {
	const defaultTags = "example,e2e,integration,functional,smoke"
	const defaultTesto = "github.com/ozontech/testo"

	Register("lint", "Run testo linter", func(f *flag.FlagSet, cmd *Lint) {
		f.StringVar(&cmd.Load.Tags, "tags", defaultTags, "build tags separated by comma")
		f.StringVar(&cmd.Load.Pkg, "pkg", defaultTesto, "testo package")
		f.BoolVar(&cmd.Load.Strict, "strict", false, "enable strict mode")
		f.BoolVar(&cmd.JSON, "json", false, "output json")
	})

	Register("suites", "Show testo suites", func(f *flag.FlagSet, cmd *Suites) {
		f.StringVar(&cmd.Load.Tags, "tags", defaultTags, "build tags separated by comma")
		f.StringVar(&cmd.Load.Pkg, "testo", defaultTesto, "testo package")
	})

	Register("version", "Show testo version", func(*flag.FlagSet, *Version) {})
}

type Version struct{}

func (cmd Version) Run(...string) error {
	version := "unknown"

	info, ok := debug.ReadBuildInfo()
	if ok {
		version = info.Main.Version
	}

	fmt.Printf("testo version %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)

	return nil
}

type Lint struct {
	Load loader.LoadSuiteConfig
	JSON bool
}

func (cmd Lint) Run(patterns ...string) error {
	_, err := loader.LoadSuites(cmd.Load, patterns...)
	if err == nil {
		return nil
	}

	var errLoad *loader.LoadError

	if errors.As(err, &errLoad) {
		w := bufio.NewWriter(os.Stdout)

		for _, d := range errLoad.Diagnostics {
			if cmd.JSON {
				d.FormatJSON(w, errLoad.FSet)
			} else {
				w.WriteString(d.Format(errLoad.FSet))
				w.WriteString("\n")
			}
		}

		w.Flush()

		os.Exit(1)
	}

	return err
}

type Suites struct {
	Load loader.LoadSuiteConfig
}

func (cmd Suites) Run(patterns ...string) error {
	suites, err := loader.LoadSuites(cmd.Load, patterns...)
	if err != nil {
		return err
	}

	for i, s := range suites {
		w := bufio.NewWriter(os.Stdout)

		fmt.Fprintln(w, "[S] "+s.Name)

		for i, t := range s.Tests {
			symbol := "└"
			fallback := " "

			if i != len(s.Tests)-1 {
				symbol = "├"
				fallback = "│"
			}

			symbol += "──"

			if t.Parametrized {
				fmt.Fprintf(w, " %s [T] %s\n", symbol, t.Name)

				for j, p := range t.Parameters {
					symbol := "└"

					if j != len(t.Parameters)-1 {
						symbol = "├"
					}

					symbol += "──"

					fmt.Fprintf(w, " %s    %s [P] %s\n", fallback, symbol, p)
				}
			} else {
				fmt.Fprintf(w, " %s [T] %s\n", symbol, t.Name)
			}
		}

		if i != len(suites)-1 {
			fmt.Fprintln(w)
		}

		w.Flush()
	}

	return nil
}
