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
// Global options are prepended to each [RunSuite] call.
//
//	func init() {
//	    testo.Options(myplugin.OutputDir("..."))
//	}
//
// It returns an empty struct to enable the following usage:
//
//	var _ = testo.Options(...)
//
// This is similar to [For] and slightly more concise than using init.
func Options(options ...testoplugin.Option) struct{} {
	globalOptionsMutex.Lock()
	defer globalOptionsMutex.Unlock()

	globalOptions = append(globalOptions, options...)

	return struct{}{}
}

func getOptions() []testoplugin.Option {
	globalOptionsMutex.RLock()
	defer globalOptionsMutex.RUnlock()

	return slices.Clone(globalOptions)
}
