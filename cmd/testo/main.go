package main

import (
	"bytes"
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"
)

func usage(f *flag.FlagSet) {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "Usage %s:\n\n", os.Args[0])

	for _, cmd := range slices.Sorted(maps.Keys(commands)) {
		fmt.Fprintf(&buf, "%-10s %s\n", cmd, commands[cmd].Desc)
	}

	f.Output().Write(buf.Bytes())
	f.PrintDefaults()
}

func main() {
	const defaultTags = "example,e2e,integration,functional,smoke"

	Register("lint", "run testo linter", func(f *flag.FlagSet, cmd *Lint) {
		f.StringVar(&cmd.Load.Tags, "tags", defaultTags, "build tags separated by comma")
		f.StringVar(&cmd.Load.Testo, "testo", "github.com/ozontech/testo", "testo package path")
		f.BoolVar(&cmd.Load.Strict, "strict", false, "enable strict mode")
		f.BoolVar(&cmd.JSON, "json", false, "output json")
	})

	Register("suites", "show testo suites", func(f *flag.FlagSet, cmd *Suites) {
		f.StringVar(&cmd.Load.Tags, "tags", defaultTags, "build tags separated by comma")
		f.StringVar(&cmd.Load.Testo, "testo", "github.com/ozontech/testo", "testo package path")
	})

	flag.Usage = func() {
		usage(flag.CommandLine)
	}

	if len(os.Args) < 2 {
		flag.Parse()
		flag.Usage()

		os.Exit(2)
	}

	if err := run(os.Args[1], os.Args[2:]...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func run(command string, args ...string) error {
	r, ok := commands[command]
	if !ok {
		usage(flag.CommandLine)
		os.Exit(2)

		return nil
	}

	return r.Run(args...)
}

type Command interface {
	Run(args ...string) error
}

var commands = make(map[string]registered)

type registered struct {
	Desc string
	Run  func(args ...string) error
}

func Register[C Command](name, desc string, flags func(f *flag.FlagSet, cmd *C)) {
	var command C

	commands[name] = registered{
		Desc: desc,
		Run: func(args ...string) error {
			f := flag.NewFlagSet(name, flag.ExitOnError)

			flags(f, &command)

			old := f.Usage

			f.Usage = func() {
				fmt.Fprintf(f.Output(), "%s\n\n", desc)

				old()
			}

			if err := f.Parse(args); err != nil {
				return err
			}

			return command.Run(f.Args()...)
		},
	}
}
