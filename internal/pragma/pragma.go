// Package pragma provides types that can be embedded into a struct to
// statically enforce or prevent certain language properties.
package pragma

// DoNotImplement can be embedded in an interface to prevent
// trivial implementations of the interface.
//
// This is useful to prevent unauthorized implementations of an
// interface so that it can be extended in the future for any changes or
// ensure certain guarantees for type conversion.
type DoNotImplement interface{ TestoInternal(_ DoNotImplement) }
