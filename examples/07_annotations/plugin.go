//go:build example

package main

import (
	"cmp"
	"slices"

	"github.com/ozontech/testo"
	"github.com/ozontech/testo/testoplugin"
	"github.com/ozontech/testo/testoreflect"
)

type Order int

const (
	First Order = iota - 1
	None
	Last
)

type utilCfg struct {
	priority Order
	runs     int
}

func newUtilCfg(options ...testoplugin.Option) utilCfg {
	cfg := utilCfg{
		priority: None,
		runs:     1,
	}

	for _, opt := range options {
		if o, ok := opt.Value.(utilOption); ok {
			o(&cfg)
		}
	}

	return cfg
}

type utilOption func(*utilCfg)

func WithOrder(priority Order) testoplugin.Option {
	return testoplugin.Option{
		Value: utilOption(func(cfg *utilCfg) { cfg.priority = priority }),
	}
}

func WithRuns(runs int) testoplugin.Option {
	return testoplugin.Option{
		Value: utilOption(func(cfg *utilCfg) { cfg.runs = runs }),
	}
}

type PluginUtils struct{ *testo.T }

func (*PluginUtils) Plugin(testoplugin.Plugin, ...testoplugin.Option) testoplugin.Spec {
	return testoplugin.Spec{
		Plan: testoplugin.Plan{
			Prepare: func(_ testoreflect.SuiteInfo, tests *[]testoplugin.PlannedTest) {
				prepared := make([]testoplugin.PlannedTest, 0, len(*tests))

				for _, t := range *tests {
					cfg := newUtilCfg(t.Annotations()...)

					for range cfg.runs {
						prepared = append(prepared, t)
					}
				}

				slices.SortStableFunc(prepared, func(a, b testoplugin.PlannedTest) int {
					aCfg := newUtilCfg(a.Annotations()...)
					bCfg := newUtilCfg(b.Annotations()...)

					return cmp.Compare(aCfg.priority, bCfg.priority)
				})

				*tests = prepared
			},
		},
	}
}
