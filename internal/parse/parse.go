// Package parse provides parsing utilities.
package parse

import "strconv"

// Bool parses string as bool treating it as false
// in case of a failure.
func Bool(s string) bool {
	b, _ := strconv.ParseBool(s)

	return b
}
