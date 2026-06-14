// Package parse provides parsing utilities.
package parse

import (
	"strconv"
	"strings"
)

// Bool parses string as bool treating it as false
// in case of a failure.
func Bool(s string) bool {
	b, _ := strconv.ParseBool(strings.TrimSpace(s))

	return b
}
