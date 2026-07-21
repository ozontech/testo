package cli

import (
	"bytes"
	"cmp"
	_ "embed"
	"errors"
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
		fmt.Fprintf(&buf, "  %-10s %s\n", cmd, commands[cmd].Short)
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
		var errExit ExitError

		if !errors.As(err, &errExit) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}

		errExit.Print()
		os.Exit(errExit.Code)
	}
}

func run(command string, args ...string) error {
	switch command {
	case "-h", "-help", "--help":
		usage(flag.CommandLine)

		return nil
	}

	r, ok := commands[command]
	if !ok {
		fmt.Fprintf(flag.CommandLine.Output(), "unknown subcommand: %q\n\n", command)

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
	Short string
	Run   func(args ...string) error
}

type config struct {
	Short string
	Usage string
	Long  string
	Args  ArgsFunc
}

type Option func(conf *config)

func WithShort(short string) Option {
	return func(conf *config) {
		conf.Short = short
	}
}

func WithUsage(u string) Option {
	return func(conf *config) {
		conf.Usage = u
	}
}

func WithLong(u string) Option {
	return func(conf *config) {
		conf.Long = u
	}
}

func WithoutArgs() Option {
	return func(conf *config) {
		conf.Args = func(args ...string) error {
			if len(args) == 0 {
				return nil
			}

			return fmt.Errorf("unexpected argument: %q", args[0])
		}
	}
}

type ArgsFunc func(args ...string) error

func Add[C Command](name string, flags func(f *flag.FlagSet, cmd *C), options ...Option) {
	var conf config

	for _, o := range options {
		o(&conf)
	}

	var command C

	commands[name] = registered{
		Short: conf.Short,
		Run: func(args ...string) error {
			f := flag.NewFlagSet(name, flag.ExitOnError)

			flags(f, &command)

			f.Usage = func() {
				long := cmp.Or(conf.Long, conf.Short)

				if long != "" {
					fmt.Fprintf(f.Output(), "%s\n\n", long)
				}

				fmt.Fprintln(f.Output(), "Usage:")

				var hasFlags bool

				f.VisitAll(func(*flag.Flag) { hasFlags = true })

				if conf.Usage != "" {
					fmt.Fprintf(f.Output(), "  %s %s %s\n", os.Args[0], name, conf.Usage)
				} else if hasFlags {
					fmt.Fprintf(f.Output(), "  %s %s [flags]\n", os.Args[0], name)
				} else {
					fmt.Fprintf(f.Output(), "  %s %s\n", os.Args[0], name)
				}

				if hasFlags {
					fmt.Fprintln(f.Output(), "\nFlags:")

					f.PrintDefaults()
				}
			}

			if err := parseFlagSet(f, args); err != nil {
				return err
			}

			positional := f.Args()

			if conf.Args != nil {
				if err := conf.Args(positional...); err != nil {
					return fmt.Errorf("%s %s: %w", os.Args[0], name, err)
				}
			}

			return command.Run(positional...)
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
