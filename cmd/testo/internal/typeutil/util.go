package typeutil

import "go/types"

func Format(t types.Type) string {
	switch t := t.(type) {
	case *types.Named:
		return formatNamed(t)

	case *types.Pointer:
		return "*" + Format(t.Elem())

	default:
		return t.String()
	}
}

func formatNamed(t *types.Named) string {
	obj := t.Obj()

	if pkg := obj.Pkg(); pkg != nil {
		return pkg.Name() + "." + obj.Name()
	}

	return obj.Name()
}
