package cmdrun

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"

	"github.com/ozontech/testo/cmd/testo/internal/cli"
	"github.com/ozontech/testo/cmd/testo/internal/cmd"
	"github.com/ozontech/testo/cmd/testo/internal/loader"
)

func init() {
	cli.Add("run", func(f *flag.FlagSet, c *Cmd) {
		f.StringVar(&c.Load.Tags, "tags", "", "build tags separated by comma, derived from source if empty")
		f.StringVar(&c.Load.Testo, "testo", cmd.DefaultTesto, "testo package")
		f.BoolVar(&c.N, "n", false, "print the commands but do not run them")
		f.BoolVar(&c.Verbose, "v", false, "verbose output")
		f.BoolVar(&c.JSON, "json", false, "log verbose output and test results in JSON")
	},
		cli.WithShort("Run testo suites"),
		cli.WithUsage(`[flags] [pattern] [flags] -- [test flags]

Patterns:
  suite              suite regex
  suite.test         suite and test regex
  .test              test regex
  package/suite      package and suite
  package/suite.test package, suite and test regex
  package/           package regex`),
	)
}

type Cmd struct {
	Load    loader.Config
	N       bool
	Verbose bool
	JSON    bool
}

type runMatched struct {
	Suite loader.Suite
	Tests map[string]struct{}
}

func (c Cmd) Run(patterns ...string) error {
	id, extraFlags, err := c.parse(patterns...)
	if err != nil {
		return err
	}

	suites, err := loader.Load(c.Load, "./...")
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

			id := s.ID()

			if m, ok := matched[id]; ok {
				maps.Copy(m.Tests, tests)
			} else {
				matched[id] = runMatched{
					Suite: s,
					Tests: tests,
				}
			}
		}
	} else {
		for _, s := range suites {
			matched[s.ID()] = runMatched{Suite: s}
		}
	}

	if len(matched) == 0 {
		if id != nil {
			return fmt.Errorf("%q did not match any suites", id.Source)
		}

		return errors.New("testo suites not found")
	}

	res, err := c.buildGoTest(slices.Collect(maps.Values(matched)), extraFlags)
	if err != nil {
		return fmt.Errorf("failed to build go test command: %w", err)
	}

	if c.N {
		fmt.Println(res.String())

		return nil
	}

	return res.Run()
}

func (c Cmd) buildGoTest(matched []runMatched, extra []string) (*exec.Cmd, error) {
	packages := make(map[string]struct{})
	suiteCallers := make(map[string]struct{})
	tests := make(map[string]struct{})

	seenTags := make(map[string]bool)
	var tags []string

	for _, m := range matched {
		runners, err := m.Suite.Runners(context.Background())
		if err != nil {
			return nil, fmt.Errorf("find runners for suite %q: %w", m.Suite.Name, err)
		}

		for _, r := range runners {
			packages[r.Dir] = struct{}{}
			suiteCallers[fmt.Sprintf("^%s$/^%s$", r.Name, m.Suite.Name)] = struct{}{}

			for t := range strings.SplitSeq(r.Tags, ",") {
				if seenTags[t] {
					continue
				}

				seenTags[t] = true
				tags = append(tags, t)
			}
		}

		for t := range m.Tests {
			tests[t] = struct{}{}
		}
	}

	if len(matched) > 0 && len(packages) == 0 {
		return nil, errors.New("suite callers not found")
	}

	args := []string{"test", "-tags", strings.Join(tags, ",")}

	if c.Verbose {
		args = append(args, "-v")
	}

	if c.JSON {
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

	command := exec.Command("go", args...)

	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Env = os.Environ()

	return command, nil
}

func (c Cmd) parse(args ...string) (id *runID, extra []string, err error) {
	for i, p := range args {
		if strings.HasPrefix(p, "-") {
			extra = append(extra, args[i:]...)

			break
		}

		if id != nil {
			return nil, nil, fmt.Errorf("unexpected positional argument: %q", p)
		}

		parsed, err := c.id(p)
		if err != nil {
			return nil, nil, fmt.Errorf("parse %q: %w", p, err)
		}

		id = &parsed
	}

	return id, extra, nil
}

func (c Cmd) id(pattern string) (runID, error) {
	id := runID{Source: pattern}

	var pkg, suite, test string

	pattern, rest, ok := strings.Cut(pattern, "/")
	if ok {
		pkg = pattern

		pattern = rest
	}

	suite, test, _ = strings.Cut(pattern, ".")

	var err error

	if pkg != "" {
		id.Package, err = regexp.Compile(pkg)
		if err != nil {
			return runID{}, fmt.Errorf("parse package %q: %w", pkg, err)
		}
	}

	if suite != "" {
		id.Suite, err = regexp.Compile(suite)
		if err != nil {
			return runID{}, fmt.Errorf("parse suite %q: %w", suite, err)
		}
	}

	if test != "" {
		id.Test, err = regexp.Compile(test)
		if err != nil {
			return runID{}, fmt.Errorf("parse test %q: %w", test, err)
		}
	}

	return id, nil
}

type runID struct {
	Package *regexp.Regexp
	Suite   *regexp.Regexp
	Test    *regexp.Regexp

	Source string
}

func (id runID) match(suite loader.Suite) (tests map[string]struct{}, ok bool) {
	if id.Package != nil && !id.Package.MatchString(suite.Package.Path) {
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
