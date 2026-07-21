package cli

import (
	"flag"
	"io"
	"slices"
	"testing"
)

func TestParseFlagSet(t *testing.T) {
	for _, tc := range []struct {
		name       string
		args       []string
		verbose    bool
		positional []string
	}{
		{
			name:       "flags only",
			args:       []string{"-v"},
			verbose:    true,
			positional: []string{},
		},
		{
			name:       "positional only",
			args:       []string{"Suite"},
			positional: []string{"Suite"},
		},
		{
			name:       "interleaved flags",
			args:       []string{"Suite", "-v", "Another"},
			verbose:    true,
			positional: []string{"Suite", "Another"},
		},
		{
			name:       "extra flags after terminator",
			args:       []string{"--", "-count=1"},
			positional: []string{"-count=1"},
		},
		{
			name:       "flags then terminator then extra flags",
			args:       []string{"-v", "--", "-count=1", "-run", "TestFoo"},
			verbose:    true,
			positional: []string{"-count=1", "-run", "TestFoo"},
		},
		{
			name:       "pattern then terminator then extra flags",
			args:       []string{"Suite", "-v", "--", "-count=1"},
			verbose:    true,
			positional: []string{"Suite", "-count=1"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := flag.NewFlagSet("test", flag.ContinueOnError)
			f.SetOutput(io.Discard)

			verbose := f.Bool("v", false, "")

			if err := parseFlagSet(f, tc.args); err != nil {
				t.Fatalf("parseFlagSet: %v", err)
			}

			if *verbose != tc.verbose {
				t.Errorf("verbose = %v, want %v", *verbose, tc.verbose)
			}

			if !slices.Equal(f.Args(), tc.positional) {
				t.Errorf("positional = %q, want %q", f.Args(), tc.positional)
			}
		})
	}
}
