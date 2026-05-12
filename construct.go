package testo

import (
	"fmt"
	"reflect"

	"github.com/ozontech/testo/internal/reflectutil"
	"github.com/ozontech/testo/internal/stack"
	"github.com/ozontech/testo/testoplugin"
)

//nolint:cyclop,funlen // this is the core of the whole framework
func construct[T CommonT](
	t TestingT,
	parent *T,
	fill func(t *testoT),
	options ...testoplugin.Option,
) T {
	t.Helper()

	seed := testoT{
		common:       t,
		testingT:     t,
		levelOptions: options,
	}

	if parent != nil {
		seed.parent = (*parent).unwrap()
	}

	if fill != nil {
		fill(&seed)
	}

	// Passed T type may be an interface.
	// In that case, we can not initialize it, since
	// we don't know its underlying concrete type.
	//
	// However, parent is always a concrete value,
	// so, if present, we extract that type from it.
	realType := reflect.TypeFor[T]()

	if parent != nil {
		rv := reflect.ValueOf(*parent)

		realType = rv.Type()
	}

	// special case when T is *testo.T
	if realType == reflect.TypeFor[*testoT]() {
		return any(&seed).(T)
	}

	value := reflect.New(realType)

	if !reflectutil.Fill(value) {
		panic(fmt.Sprintf(
			"testo: infinite type recursion detected for %s inside %s",
			reflectutil.FindRecursiveType(realType),
			realType,
		))
	}

	//nolint:nestif // TODO: factor out common logic
	if parent == nil {
		pluginTypes := make(map[reflect.Type]struct{})

		collectPlugins(realType, pluginTypes)

		delete(pluginTypes, realType)

		plugins := make(map[reflect.Type]testoplugin.Plugin, len(pluginTypes))

		for pluginType := range pluginTypes {
			var child testoplugin.Plugin

			if pluginType == reflect.TypeFor[*testoT]() {
				child = &seed
			} else {
				v := reflect.New(pluginType.Elem())

				reflectutil.Fill(v)

				child = v.Interface().(testoplugin.Plugin)
			}

			plugins[pluginType] = child
		}

		seed.plugins = plugins
	} else {
		parentUnwrapped := (*parent).unwrap()

		plugins := make(map[reflect.Type]testoplugin.Plugin, len(parentUnwrapped.plugins))

		for pluginType := range parentUnwrapped.plugins {
			var child testoplugin.Plugin

			if pluginType == reflect.TypeFor[*testoT]() {
				child = &seed
			} else {
				v := reflect.New(pluginType.Elem())

				reflectutil.Fill(v)

				child = v.Interface().(testoplugin.Plugin)
			}

			plugins[pluginType] = child
		}

		seed.plugins = plugins
	}

	specsStack := stack.New[typedPlugin]()

	for pluginType, pluginValue := range seed.plugins {
		specsStack.Push(typedPlugin{
			Plugin: pluginValue,
			Type:   pluginType,
		})

		setPlugins(reflect.ValueOf(pluginValue), seed.plugins, &specsStack)
	}

	setPlugins(value, seed.plugins, &specsStack)

	specs := make(map[reflect.Type]testoplugin.Spec, len(seed.plugins))

	for {
		p, ok := specsStack.Pop()
		if !ok {
			break
		}

		if _, ok := specs[p.Type]; ok {
			continue
		}

		var parentPlugin testoplugin.Plugin

		if parent != nil {
			parentPlugin = (*parent).unwrap().plugins[p.Type]
		} else {
			parentPlugin = reflect.New(p.Type).Elem().Interface().(testoplugin.Plugin)
		}

		specs[p.Type] = p.Plugin.Plugin(parentPlugin, seed.options()...)
	}

	seed.spec = mergeSpecs(t, mapValues(specs)...)

	return value.Elem().Interface().(T)
}

type typedPlugin struct {
	testoplugin.Plugin

	Type reflect.Type
}

func setPlugins(
	v reflect.Value,
	plugins map[reflect.Type]testoplugin.Plugin,
	specs *stack.Stack[typedPlugin],
) {
	if plugin, ok := plugins[v.Type().Elem()]; ok {
		elem := v.Elem()

		if !elem.IsValid() {
			panic(fmt.Sprintf("testo: invalid elem for %s", v.Type()))
		}

		if !elem.CanSet() {
			// TODO(metafates): add path to the field so that it is clear where error happens
			panic(fmt.Sprintf("testo: can't set value for %s", v.Type()))
		}

		elem.Set(reflect.ValueOf(plugin))

		specs.Push(typedPlugin{
			Plugin: elem.Interface().(testoplugin.Plugin),
			Type:   elem.Type(),
		})
	}

	v = reflectutil.Elem(v)

	if v.Kind() != reflect.Struct {
		return
	}

	// special case - we do not go deeper than that.
	if v.Type() == reflect.TypeFor[T]() {
		return
	}

	for i := range v.NumField() {
		field := v.Field(i)

		if !v.Type().Field(i).IsExported() {
			continue
		}

		if field.Kind() != reflect.Pointer {
			panic(
				fmt.Sprintf(
					"testo: all exported fields in T must be pointers, got: %s",
					field.Type(),
				),
			)
		}

		setPlugins(field.Addr(), plugins, specs)
	}
}

var pluginInterfaceType = reflect.TypeFor[testoplugin.Plugin]()

func collectPlugins(typ reflect.Type, plugins map[reflect.Type]struct{}) {
	if typ.Implements(pluginInterfaceType) {
		plugins[typ] = struct{}{}
	}

	typ = reflectutil.Elem(typ)

	if typ.Kind() != reflect.Struct {
		return
	}

	for i := range typ.NumField() {
		field := typ.Field(i)

		if field.IsExported() {
			collectPlugins(field.Type, plugins)
		}
	}
}

func mapValues[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m))

	for _, v := range m {
		values = append(values, v)
	}

	return values
}
