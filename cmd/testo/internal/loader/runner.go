package loader

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ozontech/testo/cmd/testo/internal/gopls"
)

func Runners(ctx context.Context, tags string, suite Suite) ([]SuiteRunner, error) {
	f := suite.FSet.File(suite.Pos)
	if f == nil {
		return nil, fmt.Errorf("suite not found in file set")
	}

	pos := f.Position(suite.Pos)

	refs, err := gopls.References(ctx, gopls.ReferencesOpts{
		Tags: tags,
		Position: gopls.Position{
			File:   f.Name(),
			Line:   pos.Line,
			Column: pos.Column,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("gopls references: %w", err)
	}

	var runners []SuiteRunner

	for _, r := range refs {
		runner, ok := asRunner(r, suite)
		if !ok {
			continue
		}

		runners = append(runners, runner)
	}

	return runners, nil
}

func asRunner(ref gopls.Position, suite Suite) (SuiteRunner, bool) {
	if !strings.HasSuffix(ref.File, "_test.go") {
		return SuiteRunner{}, false
	}

	file, err := os.Open(ref.File)
	if err != nil {
		return SuiteRunner{}, false
	}
	defer file.Close()

	s := bufio.NewScanner(file)

	var (
		i    int
		test string
	)

	for s.Scan() {
		i++

		line := s.Text()

		if strings.HasPrefix(line, "func Test") {
			before, _, _ := strings.Cut(line, "(")

			test = strings.Fields(before)[1]
		}

		if i != ref.Line {
			continue
		}

		if strings.Contains(line, "testo.RunSuite") {
			return SuiteRunner{
				Dir:  filepath.Dir(ref.File),
				Name: test,
			}, true
		}
	}

	err = s.Err()
	if err != nil {
		return SuiteRunner{}, false
	}

	fmt.Printf("ref: %#+v\n", ref)
	// TODO

	return SuiteRunner{}, false
}
