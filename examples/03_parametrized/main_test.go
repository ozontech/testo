//go:build example

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ozontech/testo"
)

func Test(t *testing.T) {
	testo.RunSuite(t, new(Suite))
}

type T = *testo.T

type Suite struct {
	testo.Suite[T]

	path string
}

func (s *Suite) BeforeEach(t T) {
	s.path = createSimpleGoProgram(t)
}

func (*Suite) CasesOS() []string {
	return []string{"linux", "windows", "darwin"}
}

func (*Suite) CasesArch() []string {
	return []string{"amd64", "arm64"}
}

func (s *Suite) TestCompile(t T, p struct{ OS, Arch string }) {
	t.Logf("building for %s/%s", p.OS, p.Arch)

	cmd := exec.Command("go", "build", s.path)
	cmd.Dir = filepath.Dir(s.path)
	cmd.Env = append(os.Environ(), "GOOS="+p.OS, "GOARCH="+p.Arch)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("failed to compile for %s/%s: %q", p.OS, p.Arch, string(output))
	}
}

func createSimpleGoProgram(t T) string {
	t.Helper()

	dir := filepath.Join(t.TempDir(), "testo-example-compile")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal("failed to created temp dir", err)
	}

	path := filepath.Join(dir, "main.go")

	if err := os.WriteFile(
		path,
		[]byte(`package main; func main() { println("hello world") }`),
		0o600,
	); err != nil {
		t.Fatal("failed to write main.go", err)
	}

	return path
}
