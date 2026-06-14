package testo

import (
	"flag"
	"os"
	"regexp"

	"github.com/ozontech/testo/internal/parse"
	// so that cache flags are always available.
	_ "github.com/ozontech/testo/testocache"
)

var flagStrict = flag.Bool(
	"testo.strict",
	parse.Bool(os.Getenv("TESTO_STRICT")),
	"run tests in strict mode",
)

var flagMethod = flagRegexp{Regexp: regexp.MustCompile("")}

func init() {
	flag.Var(&flagMethod, "testo.m", "regular expression to select tests of the testo suite to run")
}

var _ flag.Value = (*flagRegexp)(nil)

type flagRegexp struct{ *regexp.Regexp }

func (f *flagRegexp) Set(s string) error {
	exp, err := regexp.Compile(s)
	if err != nil {
		return err
	}

	f.Regexp = exp

	return nil
}

func (f *flagRegexp) String() string {
	if f.Regexp == nil {
		return ""
	}

	return f.Regexp.String()
}
