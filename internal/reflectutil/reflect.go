package reflectutil

import (
	"reflect"
)

type canElem[Self any] interface {
	Kind() reflect.Kind
	Elem() Self
}

// Elem unwraps the underlying elem of the pointer.
//
// Nested pointers are also supported - e.g. given "****value" it will return "value".
//
// Non-pointer values will be returned as is.
func Elem[T canElem[T]](v T) T {
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	return v
}

// NameOf returns name of the underlying type T.
func NameOf[T any]() string {
	t := reflect.TypeFor[T]()

	return Elem(t).Name()
}

// New a new zero value of T.
//
// As a special case for pointers it will
// return pointer to the zero value of T (not nil).
func New[T any]() T {
	t := reflect.TypeFor[T]()

	var zero T

	if t.Kind() == reflect.Pointer {
		elem := reflect.ValueOf(&zero).Elem()

		elem.Set(reflect.New(t.Elem()))
	}

	return zero
}

// Filled returns a new value T with all the exported pointer fields recursively set to non-nil zero values.
// That is, if type is a struct and contains field *int it will be set to &0.
// That logic is also applies for nested structs.
func Filled(hint reflect.Type) (reflect.Value, bool) {
	value := reflect.New(hint)

	ok := Fill(value)

	return value.Elem(), ok
}

func Fill(v reflect.Value) bool {
	return fill(v, make(map[reflect.Type]bool))
}

func FindRecursiveType(t reflect.Type) reflect.Type {
	return findRecursiveType(t, make(map[reflect.Type]bool))
}

func findRecursiveType(t reflect.Type, visited map[reflect.Type]bool) reflect.Type {
	switch t.Kind() {
	case reflect.Pointer:
		return findRecursiveType(t.Elem(), visited)

	case reflect.Struct:
		if visited[t] {
			return t
		}

		visited[t] = true

		for i := range t.NumField() {
			field := t.Field(i)

			if !field.IsExported() {
				continue
			}

			if found := findRecursiveType(field.Type, visited); found != nil {
				return found
			}
		}

		visited[t] = false

		return nil

	default:
		return nil
	}
}

// fill returns true on success, meaning that type recursion was not detected.
func fill(v reflect.Value, visited map[reflect.Type]bool) bool {
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		return fill(v.Elem(), visited)

	case reflect.Struct:
		typ := v.Type()

		if visited[typ] {
			return false
		}

		visited[typ] = true

		for i := range v.NumField() {
			field := v.Field(i)

			if !field.CanSet() {
				continue
			}

			if !fill(field, visited) {
				return false
			}
		}

		visited[typ] = false

		return true

	default:
		return true
	}
}
