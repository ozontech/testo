package env

import (
	"os"
	"testing"
)

func TestEnv(t *testing.T) {
	const Var Env = "TEST_EXAMPLE"
	const value = "lorem ipsum"

	t.Setenv(string(Var), value)

	if Var.Get() != os.Getenv(string(Var)) {
		t.Fatal("Env.Get must be equal to os.Getenv")
	}
}
