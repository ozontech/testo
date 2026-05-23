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
//	func Test(t *testing.T) {
//		testo.RunTest(t, func(t *testo.T) {
//			t.Log("Hello, Testo!")
//		})
//	}
//
// # Plugins
//
// Plugins are the core feature of Testo.
// Plugins can generate reports, add custom methods to T,
// override built-in methods, plan test execution and more.
//
// Plugins are installed by defining our own T with embedded [T] and plugins:
//
//	type T struct {
//		*testo.T
//		*myplugin.PluginFoo
//		*myplugin.PluginBar
//	}
//
//	func Test(t *testing.T) {
//		testo.RunTest(t, func(t T) {
//			t.Log("Hello, Testo!")
//		})
//	}
//
// Notice that we now use our T instead of *testo.T in a test.
//
// # Suites & Standalone Tests
//
// Testo supports several ways to run tests.
//
// Suites:
//
//	type Suite struct { testo.Suite[T] }
//
//	func (Suite) TestFoo(t T) { t.Log("Foo") }
//	func (Suite) TestBar(t T) { t.Log("Bar") }
//
//	func Test(t *testing.T) {
//		testo.RunSuite(t, new(Suite))
//	}
//
// Standalone:
//
//	func TestFoo(t *testing.T) {
//		testo.RunTest(t, func(t T) {
//			t.Log("Foo")
//		})
//	}
//
//	func Test(t *testing.T) {
//		t.Run("Foo", testo.Test(func(t T) {
//			t.Log("Foo")
//		}))
//
//		t.Run("Bar", testo.Test(func(t T) {
//			t.Log("Bar")
//		}))
//	}
package testo
