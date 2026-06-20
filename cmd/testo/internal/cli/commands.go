package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/ozontech/testo/cmd/testo/internal/loader"
)

func init() {
	const defaultTags = "example,e2e,integration,functional,smoke"
	const defaultTesto = "github.com/ozontech/testo"

	Register("lint", "Run testo linter", func(f *flag.FlagSet, cmd *Lint) {
		f.StringVar(&cmd.Load.Tags, "tags", defaultTags, "build tags separated by comma")
		f.StringVar(&cmd.Load.Testo, "testo", defaultTesto, "testo package")
		f.BoolVar(&cmd.Load.Strict, "strict", false, "enable strict mode")
		f.BoolVar(&cmd.JSON, "json", false, "output json")
	})

	Register("run", "Run testo suites", func(f *flag.FlagSet, cmd *RunCmd) {
		f.StringVar(&cmd.Load.Tags, "tags", defaultTags, "build tags separated by comma")
		f.StringVar(&cmd.Load.Testo, "testo", defaultTesto, "testo package")
		f.StringVar(&cmd.Sep, "s", ".", "identifier delimieter")
	})

	Register("suites", "Show testo suites", func(f *flag.FlagSet, cmd *Suites) {
		f.StringVar(&cmd.Load.Tags, "tags", defaultTags, "build tags separated by comma")
		f.StringVar(&cmd.Load.Testo, "testo", defaultTesto, "testo package")
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
	Load loader.Config
	JSON bool
}

func (cmd Lint) Run(patterns ...string) error {
	_, err := loader.Load(cmd.Load, patterns...)
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
	Load loader.Config
}

func (cmd Suites) Run(patterns ...string) error {
	suites, err := loader.Load(cmd.Load, patterns...)
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

type RunCmd struct {
	Load loader.Config
	Sep  string
}

func (cmd RunCmd) Run(patterns ...string) error {
	ids, err := cmd.ids(patterns...)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return errors.New("at least one pattern is required")
	}

	suites, err := loader.Load(cmd.Load, "./...")
	if err != nil {
		return err
	}

	type Matched struct {
		Suite loader.Suite
		Tests map[string]struct{}
	}

	matched := make(map[string]Matched)

	for _, s := range suites {
		for _, id := range ids {
			tests, ok := id.match(s)

			if !ok {
				continue
			}

			key := s.Package + "." + s.Name

			if m, ok := matched[key]; ok {
				maps.Copy(m.Tests, tests)
			} else {
				matched[key] = Matched{
					Suite: s,
					Tests: tests,
				}
			}
		}
	}

	if len(matched) == 0 {
		return errors.New("no suites matched")
	}

	var suiteRunners []string
	var tests []string

	for _, m := range matched {
		suiteRunners = append(suiteRunners, "Test"+m.Suite.Name)

		for t := range m.Tests {
			tests = append(tests, t)
		}
	}

	args := []string{
		"go",
		"test",
		"./...",
		"-run",
		fmt.Sprintf("^(%s)$", strings.Join(suiteRunners, "|")),
	}

	if len(tests) > 0 {
		args = append(
			args,
			"-testo.m",
			fmt.Sprintf("^(%s)$", strings.Join(tests, "|")),
		)
	}

	fmt.Println(strings.Join(args, " "))

	return nil
}

func (cmd RunCmd) ids(patterns ...string) ([]runID, error) {
	ids := make([]runID, 0, len(patterns))

	for _, p := range patterns {
		parsed, err := cmd.id(p)
		if err != nil {
			return nil, fmt.Errorf("parse %q: %w", p, err)
		}

		ids = append(ids, parsed...)
	}

	return ids, nil
}

func (cmd RunCmd) id(pattern string) ([]runID, error) {
	fields := strings.Split(pattern, cmd.Sep)

	switch len(fields) {
	case 1: // suite
		suite, err := regexp.Compile(fields[0])
		if err != nil {
			return nil, err
		}

		return []runID{{
			Suite: suite,
		}}, nil

	case 2: // package.suite || suite.test
		first, err := regexp.Compile(fields[0])
		if err != nil {
			return nil, err
		}

		second, err := regexp.Compile(fields[1])
		if err != nil {
			return nil, err
		}

		return []runID{
			{
				Package: first,
				Suite:   second,
			},
			{
				Suite: first,
				Test:  second,
			},
		}, nil

	case 3: // package.suite.test
		pkg, err := regexp.Compile(fields[0])
		if err != nil {
			return nil, err
		}

		suite, err := regexp.Compile(fields[1])
		if err != nil {
			return nil, err
		}

		test, err := regexp.Compile(fields[2])
		if err != nil {
			return nil, err
		}

		return []runID{{
			Package: pkg,
			Suite:   suite,
			Test:    test,
		}}, nil

	default:
		return nil, errors.New("invalid syntax")
	}
}

type runID struct {
	Package *regexp.Regexp
	Suite   *regexp.Regexp
	Test    *regexp.Regexp
}

func (id runID) match(suite loader.Suite) (tests map[string]struct{}, ok bool) {
	if id.Package != nil && !id.Package.MatchString(suite.Package) {
		return nil, false
	}

	if id.Suite != nil && !id.Suite.MatchString(suite.Name) {
		return nil, false
	}

	if id.Test == nil {
		return nil, true
	}

	tests = make(map[string]struct{})

	for _, t := range suite.Tests {
		if id.Test.MatchString(t.Name) {
			tests[t.Name] = struct{}{}
			ok = true
		}
	}

	return tests, ok
}
