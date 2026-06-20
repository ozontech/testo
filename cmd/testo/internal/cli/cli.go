package cli

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"
)

//go:embed logo.txt
var logo string

func usage(f *flag.FlagSet) {
	var buf bytes.Buffer

	fmt.Fprint(&buf, logo)
	fmt.Fprintln(&buf, "Usage:")
	fmt.Fprintf(&buf, "  %s [command]\n\n", os.Args[0])
	fmt.Fprintln(&buf, "Available Commands:")

	for _, cmd := range slices.Sorted(maps.Keys(commands)) {
		fmt.Fprintf(&buf, "  %-10s %s\n", cmd, commands[cmd].Desc)
	}

	f.Output().Write(buf.Bytes())
	f.PrintDefaults()
}

func Run() {
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
		fmt.Fprintf(flag.CommandLine.Output(), "unknown subcommand %q\n\n", command)

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

func Register[C Command](name, desc, usage string, flags func(f *flag.FlagSet, cmd *C)) {
	var command C

	commands[name] = registered{
		Desc: desc,
		Run: func(args ...string) error {
			f := flag.NewFlagSet(name, flag.ExitOnError)

			flags(f, &command)

			f.Usage = func() {
				fmt.Fprintf(f.Output(), "%s\n\n", desc)
				fmt.Fprintln(f.Output(), "Usage:")

				var hasFlags bool

				f.VisitAll(func(*flag.Flag) { hasFlags = true })

				fmt.Fprintf(f.Output(), "  %s %s %s\n", os.Args[0], name, usage)

				if hasFlags {
					fmt.Fprintln(f.Output(), "\nFlags:")
					f.PrintDefaults()
				}
			}

			if err := parseFlagSet(f, args); err != nil {
				return err
			}

			return command.Run(f.Args()...)
		},
	}
}

func parseFlagSet(f *flag.FlagSet, args []string) error {
	positional := make([]string, 0, len(args))

	for {
		if err := f.Parse(args); err != nil {
			return err
		}

		args = args[len(args)-f.NArg():]
		if len(args) == 0 {
			break
		}

		positional = append(positional, args[0])

		args = args[1:]
	}

	return f.Parse(positional)
}
