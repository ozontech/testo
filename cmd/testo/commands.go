package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
)

type Lint struct {
	Load LoadSuiteConfig
}

func (cmd Lint) Run(patterns ...string) error {
	_, err := LoadSuites(cmd.Load, patterns...)
	if err == nil {
		return nil
	}

	if errLoad, ok := errors.AsType[*LoadError](err); ok {
		w := bufio.NewWriter(os.Stdout)

		type Node struct {
			File    string
			Line    int
			Kind    string
			Message string
		}

		for _, d := range errLoad.Diagnostics {
			w.WriteString(d.Format(errLoad.FSet))
			w.WriteString("\n")
		}

		w.Flush()

		os.Exit(1)
	}

	return err
}

type Suites struct {
	Load LoadSuiteConfig
}

func (cmd Suites) Run(patterns ...string) error {
	suites, err := LoadSuites(cmd.Load, patterns...)
	if err != nil {
		return err
	}

	for i, s := range suites {
		w := bufio.NewWriter(os.Stdout)

		fmt.Fprintln(w, "[S] "+s.Name)

		for i, t := range s.Tests {
			symbol := "└"
			fallback := " "

			if i != len(s.Tests)-1 {
				symbol = "├"
				fallback = "│"
			}

			symbol += "──"

			if t.Parametrized {
				fmt.Fprintf(w, " %s [T] %s\n", symbol, t.Name)

				for j, p := range t.Parameters {
					symbol := "└"

					if j != len(t.Parameters)-1 {
						symbol = "├"
					}

					symbol += "──"

					fmt.Fprintf(w, " %s    %s [P] %s\n", fallback, symbol, p)
				}
			} else {
				fmt.Fprintf(w, " %s [T] %s\n", symbol, t.Name)
			}
		}

		if i != len(suites)-1 {
			fmt.Fprintln(w)
		}

		w.Flush()
	}

	return nil
}
