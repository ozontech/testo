package typeutil

import "go/types"

func Format(t types.Type) string {
	switch t := t.(type) {
	case *types.Named:
		return formatNamedType(t)

	case *types.Pointer:
		return "*" + Format(t.Elem())

	default:
		return t.String()
	}
}

func formatNamedType(t *types.Named) string {
	pkg := t.Obj().Pkg().Name()
	name := t.Obj().Name()

	return pkg + "." + name
}
