//go:build e2e

package testo

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var ignoreLineOrder = []string{
	"02_hooks",
	"05_parallel",
}

func TestExamples(t *testing.T) {
	examples := collectExamples(t)

	for _, path := range examples {
		name := filepath.Base(path)

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			wantOutput := readFile(t, filepath.Join(path, "output.golden"))
			haveOutput := runTest(t, path)

			wantOutput = unifyOutput(wantOutput)
			haveOutput = unifyOutput(haveOutput)

			requireEqualLines(
				t,
				string(wantOutput),
				string(haveOutput),
				slices.Contains(ignoreLineOrder, name),
			)
		})
	}
}

func lines(s string) []string {
	// We store test output in files with UNIX line endings.
	// But, since windows uses CR-LF line endings, we have to rebuild these strings
	// to make line endings the same for further comparison.
	scanner := bufio.NewScanner(strings.NewReader(s))

	var lines []string

	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}

	return lines
}

func requireEqualLines(t *testing.T, a, b string, ignoreOrder bool) {
	t.Helper()

	aLines, bLines := lines(a), lines(b)

	if ignoreOrder {
		slices.Sort(aLines)
		slices.Sort(bLines)
	}

	if !slices.Equal(aLines, bLines) {
		t.Fatalf(
			"lines not equal:\n\n%s\n\n%s",
			strings.Join(aLines, "\n"),
			strings.Join(bLines, "\n"),
		)
	}
}

func collectExamples(t *testing.T) []string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	examplesDir := filepath.Join(wd, "examples")

	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatal(err)
	}

	examples := make([]string, 0, len(entries))

	namePattern := regexp.MustCompile(`^\d\d_.+$`)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		if !namePattern.MatchString(name) {
			continue
		}

		examples = append(examples, filepath.Join(examplesDir, name))
	}

	return examples
}

func unifyOutput(b []byte) []byte {
	b = replaceDurations(b, []byte("?"))
	b = replaceInners(b, []byte("inner_?"))
	b = bytes.TrimSpace(b)

	return b
}

func replaceInners(src, repl []byte) []byte {
	re := regexp.MustCompile(`inner_\d`)

	return re.ReplaceAll(src, repl)
}

func replaceDurations(src, repl []byte) []byte {
	re := regexp.MustCompile(`[+-]?(?:\d+(?:\.\d+)?(?:ns|us|µs|ms|s|m|h))+`)

	return re.ReplaceAll(src, repl)
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return contents
}

func runTest(t *testing.T, pkgPath string) []byte {
	t.Helper()

	cmd := exec.Command(
		"go",
		"test",
		"-tags",
		"example",
		"-v",
		"-count=1",    // ignore cache and always run
		"-parallel=1", // ensure output remains the same
		pkgPath,
	)

	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		var errExec *exec.ExitError

		if !errors.As(err, &errExec) || errExec.ExitCode() != 1 {
			t.Fatalf("running go test: %v", err)
		}
	}

	return output
}
