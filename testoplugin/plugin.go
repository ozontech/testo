// Package testoplugin provides plugin primitives for using plugins in testo.
//
// # Implementing a plugin
//
// Plugins can implement [Plugin] interface to be registered as such.
//
// Method "Plugin" will be called for each plugin before running a suite.
// For sub-tests, parent refers to the plugin instance of the parent test.
// For top-level tests, parent is a typed-nil instance of the plugin's own
// type: the interface itself is non-nil, but the concrete pointer is nil,
// so an unconditional type assertion like parent.(*PluginFoo) is safe.
// To detect the top level, nil-check the asserted pointer, not the interface:
//
//	prev, _ := parent.(*PluginFoo)
//	if prev != nil {
//		// sub-test: use prev
//	}
//
// It is encouraged to ensure a plugin implements [Plugin] interface with the following line:
//
//	var _ testoplugin.Plugin = (*PluginFoo)(nil)
package testoplugin

import "math"

// Priority defines execution order (priority).
// It defines when a plugin component should be
// invoked when other parts are available.
//
// "Plugin component" means plan, hook, override, etc.
//
// See [TryFirst] and [TryLast] for predefined priority values.
type Priority int

const (
	// TryFirst indicates that this plugin part should be run as early as possible.
	TryFirst Priority = math.MinInt

	// TryLast indicates that this plugin part should be run as late as possible.
	TryLast Priority = math.MaxInt
)

// Option is used to configure plugin upon creation.
//
// All user-supplied options are passed to the Plugin method for each plugin.
// It is a plugin responsibility to check if the given option corresponds to it.
// One way to check it is with type assertion:
//
//	var opt Option
//	o, ok := opt.Value.(MyPluginOption)
type Option struct {
	// Value of this option.
	Value any

	// Propagate states whether this option
	// should be passed automatically to all subtests.
	Propagate bool
}

// Propagated returns a shallow clone of this option
// with "Propagate" field set to true.
func (o Option) Propagated() Option {
	o.Propagate = true

	return o
}

// Plugin is an interface that plugins implement to provide
// [Plan], [Hooks] and [Overrides] to the tests.
type Plugin interface {
	Plugin(parent Plugin, options ...Option) Spec
}

// Spec is a plugin specification.
type Spec struct {
	Plan      Plan
	Hooks     Hooks
	Overrides Overrides
}
