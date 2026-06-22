package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/ozontech/testo/cmd/testo/internal/cli"
	"github.com/ozontech/testo/cmd/testo/internal/loader"
	"github.com/ozontech/testo/cmd/testo/internal/typeutil"
)

func main() {
	cli.Run()
}

func init() {
	const defaultTags = "example,e2e,integration,functional,smoke"
	const defaultTesto = "github.com/ozontech/testo"

	cli.Add("lint", func(f *flag.FlagSet, cmd *Lint) {
		f.StringVar(&cmd.Load.Tags, "tags", defaultTags, "build tags separated by comma")
		f.StringVar(&cmd.Load.Testo, "testo", defaultTesto, "testo package")
		f.BoolVar(&cmd.Load.Strict, "strict", false, "enable strict mode")
		f.BoolVar(&cmd.JSON, "json", false, "output json")
	}, cli.WithShort("Run testo linter"))

	cli.Add("run", func(f *flag.FlagSet, cmd *Run) {
		f.StringVar(&cmd.Load.Tags, "tags", defaultTags, "build tags separated by comma")
		f.StringVar(&cmd.Load.Testo, "testo", defaultTesto, "testo package")
		f.StringVar(&cmd.Sep, "s", ".", "pattern separator")
		f.BoolVar(&cmd.N, "n", false, "print the commands but do not run them")
		f.BoolVar(&cmd.Verbose, "v", false, "verbose output")
		f.BoolVar(&cmd.JSON, "json", false, "log verbose output and test results in JSON")
	},
		cli.WithShort("Run testo suites"),
		cli.WithUsage(`[flags] [pattern] [flags] -- [test flags]

Pattern:
  suite              suite regex
  suite.test         suite and test regex
  package.suite.test package, suite and test regex`),
	)

	cli.Add("suites", func(f *flag.FlagSet, cmd *Suites) {
		f.StringVar(&cmd.Load.Tags, "tags", defaultTags, "build tags separated by comma")
		f.StringVar(&cmd.Load.Testo, "testo", defaultTesto, "testo package")
	}, cli.WithShort("Show testo suites"))

	cli.Add(
		"version",
		func(*flag.FlagSet, *Version) {},
		cli.WithShort("Show testo version"),
		cli.WithoutArgs(),
	)
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

	if !errors.As(err, &errLoad) {
		return err
	}

	var buf bytes.Buffer

	for _, d := range errLoad.Diagnostics {
		if cmd.JSON {
			d.JSON(&buf, errLoad.FSet)
		} else {
			d.Print(&buf, errLoad.FSet)
		}
	}

	return cli.Exit(1).Stdout(buf.String())
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

					fmt.Fprintf(w, " %s    %s [P] %s (%s)\n", fallback, symbol, p.Name, typeutil.Format(p.Type))
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

type Run struct {
	Load    loader.Config
	Sep     string
	N       bool
	Verbose bool
	JSON    bool
}

type runMatched struct {
	Suite loader.Suite
	Tests map[string]struct{}
}

func (cmd Run) Run(patterns ...string) error {
	id, extraFlags, err := cmd.parsePositional(patterns...)
	if err != nil {
		return err
	}

	suites, err := loader.Load(cmd.Load, "./...")
	if err != nil {
		return err
	}

	matched := make(map[string]runMatched)

	if id != nil {
		for _, s := range suites {
			tests, ok := id.match(s)

			if !ok {
				continue
			}

			key := s.Package + "." + s.Name

			if m, ok := matched[key]; ok {
				maps.Copy(m.Tests, tests)
			} else {
				matched[key] = runMatched{
					Suite: s,
					Tests: tests,
				}
			}
		}
	} else {
		for _, s := range suites {
			key := s.Package + "." + s.Name

			matched[key] = runMatched{Suite: s}
		}
	}

	if len(matched) == 0 {
		if id != nil {
			return fmt.Errorf("%q did not match any suites", id.Source)
		}

		return errors.New("testo suites not found")
	}

	c, err := cmd.buildGoTest(slices.Collect(maps.Values(matched)), extraFlags)
	if err != nil {
		return fmt.Errorf("failed to build go test command: %w", err)
	}

	if cmd.N {
		fmt.Println(c.String())

		return nil
	}

	return c.Run()
}

func (cmd Run) buildGoTest(matched []runMatched, extra []string) (*exec.Cmd, error) {
	packages := make(map[string]struct{})
	suiteCallers := make(map[string]struct{})
	tests := make(map[string]struct{})

	for _, m := range matched {
		for _, r := range m.Suite.Runners {
			packages[r.Dir] = struct{}{}
			suiteCallers[fmt.Sprintf("^%s$/^%s$", r.Name, m.Suite.Name)] = struct{}{}
		}

		for t := range m.Tests {
			tests[t] = struct{}{}
		}
	}

	if len(matched) > 0 && len(packages) == 0 {
		return nil, errors.New("suite callers not found")
	}

	args := []string{"test", "-tags", cmd.Load.Tags}

	if cmd.Verbose {
		args = append(args, "-v")
	}

	if cmd.JSON {
		args = append(args, "-json")
	}

	for p := range packages {
		args = append(args, p)
	}

	if len(packages) == 0 {
		args = append(args, ".")
	}

	if len(suiteCallers) > 0 {
		args = append(
			args,
			"-run",
			strings.Join(slices.Sorted(maps.Keys(suiteCallers)), "|"),
		)
	}

	if len(tests) > 0 {
		args = append(
			args,
			"-testo.m",
			fmt.Sprintf(
				"^(%s)$",
				strings.Join(slices.Sorted(maps.Keys(tests)), "|"),
			),
		)
	}

	args = append(args, extra...)

	c := exec.Command("go", args...)

	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = os.Environ()

	return c, nil
}

func (cmd Run) parsePositional(args ...string) (id *runID, extra []string, err error) {
	for i, p := range args {
		if strings.HasPrefix(p, "-") {
			extra = append(extra, args[i:]...)

			break
		}

		if id != nil {
			return nil, nil, fmt.Errorf("unexpected positional argument: %q", p)
		}

		parsed, err := cmd.id(p)
		if err != nil {
			return nil, nil, fmt.Errorf("parse %q: %w", p, err)
		}

		id = &parsed
	}

	return id, extra, nil
}

func (cmd Run) id(pattern string) (runID, error) {
	fields := strings.Split(pattern, cmd.Sep)

	switch len(fields) {
	case 1: // suite
		suite, err := regexp.Compile(fields[0])
		if err != nil {
			return runID{}, err
		}

		return runID{
			Suite:  suite,
			Source: pattern,
		}, nil

	case 2: // suite.test
		suite, err := regexp.Compile(fields[0])
		if err != nil {
			return runID{}, err
		}

		test, err := regexp.Compile(fields[1])
		if err != nil {
			return runID{}, err
		}

		return runID{
			Suite:  suite,
			Test:   test,
			Source: pattern,
		}, nil

	case 3: // package.suite.test
		pkg, err := regexp.Compile(fields[0])
		if err != nil {
			return runID{}, err
		}

		suite, err := regexp.Compile(fields[1])
		if err != nil {
			return runID{}, err
		}

		test, err := regexp.Compile(fields[2])
		if err != nil {
			return runID{}, err
		}

		return runID{
			Package: pkg,
			Suite:   suite,
			Test:    test,
			Source:  pattern,
		}, nil

	default:
		return runID{}, errors.New("invalid syntax")
	}
}

type runID struct {
	Package *regexp.Regexp
	Suite   *regexp.Regexp
	Test    *regexp.Regexp

	Source string
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
