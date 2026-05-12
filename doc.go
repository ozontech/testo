// Package testo is a modular testing framework built on top of [testing.T].
// It is focused on suite based tests and has an extensive plugin system.
//
// # Quick Start
//
// A minimal working example looks like this:
//
//	package main
//
//	import (
//		"testing"
//
//		"github.com/ozontech/testo"
//	)
//
//	type T struct { *testo.T }
//	type Suite struct { testo.Suite[T] }
//
//	func (Suite) Test(t T) { t.Log("Hello, world!") }
//
//	func Test(t *testing.T) { testo.RunSuite(t, new(Suite)) }
//
// Notice the `Test(t *testing.T)` - since [RunSuite] requires
// an instance of `testing.T` we must declare a regular go test first,
// and only inside it we will be able to actually call our Testo suite.
package testo
