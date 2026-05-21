package testo

import (
	"slices"
	"sync"

	"github.com/ozontech/testo/testoplugin"
)

// Avoid using these values directly.
// Use [Options] and [getDefaultOptions] instead.
var (
	globalOptions      []testoplugin.Option
	globalOptionsMutex sync.RWMutex
)

// Options appends given options to the global options.
//
// Global options are prepended to each [RunSuite] & [RunTest] call.
//
//	func init() {
//	    testo.Options(myplugin.OutputDir("..."))
//	}
func Options(options ...testoplugin.Option) {
	globalOptionsMutex.Lock()
	defer globalOptionsMutex.Unlock()

	globalOptions = append(globalOptions, options...)
}

func getOptions() []testoplugin.Option {
	globalOptionsMutex.RLock()
	defer globalOptionsMutex.RUnlock()

	return slices.Clone(globalOptions)
}
