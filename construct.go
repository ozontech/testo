package testo

import (
	"fmt"
	"reflect"
	"slices"

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

	if parent != nil {
		seed.pluginOrder = (*parent).unwrap().pluginOrder
	} else {
		seen := make(map[reflect.Type]struct{})

		collectPlugins(realType, seen, &seed.pluginOrder)

		// the T type itself is not a plugin on its own.
		seed.pluginOrder = slices.DeleteFunc(seed.pluginOrder, func(typ reflect.Type) bool {
			return typ == realType
		})
	}

	plugins := make(map[reflect.Type]testoplugin.Plugin, len(seed.pluginOrder))

	for _, pluginType := range seed.pluginOrder {
		var child testoplugin.Plugin

		if pluginType == reflect.TypeFor[*testoT]() {
			child = &seed
		} else {
			v := reflect.New(pluginType.Elem())

			reflectutil.Fill(v)

			child = reflectutil.MustTypeAssert[testoplugin.Plugin](v)
		}

		plugins[pluginType] = child
	}

	seed.plugins = plugins

	specsStack := stack.New[typedPlugin]()

	for _, pluginType := range seed.pluginOrder {
		specsStack.Push(typedPlugin{
			Plugin: seed.plugins[pluginType],
			Type:   pluginType,
		})

		setPlugins(reflect.ValueOf(seed.plugins[pluginType]), seed.plugins, &specsStack)
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

		// For top-level tests the parent is a typed-nil instance of the
		// plugin's own type: the interface is non-nil, so unconditional
		// parent.(*MyPlugin) assertions keep working, and the concrete
		// pointer is nil. See [testoplugin.Plugin].
		var parentPlugin testoplugin.Plugin

		if parent != nil {
			parentPlugin = (*parent).unwrap().plugins[p.Type]
		} else {
			parentPlugin = reflectutil.MustTypeAssert[testoplugin.Plugin](
				reflect.New(p.Type).Elem(),
			)
		}

		specs[p.Type] = p.Plugin.Plugin(parentPlugin, seed.options()...)
	}

	// merge specs in the declaration order of plugins inside T so that
	// equal-priority hooks and overrides run deterministically.
	orderedSpecs := make([]testoplugin.Spec, 0, len(specs))

	for _, pluginType := range seed.pluginOrder {
		if spec, ok := specs[pluginType]; ok {
			orderedSpecs = append(orderedSpecs, spec)
		}
	}

	seed.spec = mergeSpecs(t, orderedSpecs...)

	return reflectutil.MustTypeAssert[T](value.Elem())
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
			// TODO(metafates): add path to the field so that it is clear where the error happens
			panic(fmt.Sprintf("testo: can't set value for %s", v.Type()))
		}

		elem.Set(reflect.ValueOf(plugin))

		specs.Push(typedPlugin{
			Plugin: reflectutil.MustTypeAssert[testoplugin.Plugin](elem),
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
					"testo: all exported fields in T and plugins must be pointers: field %s.%s has non-pointer type %s",
					v.Type(),
					v.Type().Field(i).Name,
					field.Type(),
				),
			)
		}

		setPlugins(field.Addr(), plugins, specs)
	}
}

var pluginInterfaceType = reflect.TypeFor[testoplugin.Plugin]()

// collectPlugins gathers plugin types in their declaration order:
// a depth-first walk over the exported fields of typ.
func collectPlugins(typ reflect.Type, seen map[reflect.Type]struct{}, order *[]reflect.Type) {
	if typ.Implements(pluginInterfaceType) {
		if _, ok := seen[typ]; !ok {
			seen[typ] = struct{}{}

			*order = append(*order, typ)
		}
	}

	typ = reflectutil.Elem(typ)

	if typ.Kind() != reflect.Struct {
		return
	}

	for i := range typ.NumField() {
		field := typ.Field(i)

		if field.IsExported() {
			collectPlugins(field.Type, seen, order)
		}
	}
}
