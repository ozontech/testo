package cli

import (
	"fmt"
	"os"
)

type ExitError struct {
	Code int

	stdout string
	stderr string
}

func (e ExitError) Error() string {
	return fmt.Sprintf("exit code %d", e.Code)
}

func Exit(code int) *ExitError {
	return &ExitError{
		Code: code,
	}
}

func (e ExitError) Stdout(s string) ExitError {
	e.stdout = s

	return e
}

func (e ExitError) Print() {
	if e.stdout != "" {
		fmt.Fprint(os.Stdout, e.stdout)
	}

	if e.stderr != "" {
		fmt.Fprint(os.Stderr, e.stderr)
	}
}
