// Package parse provides parsing utilities.
package parse

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Bool parses string as bool treating it as false
// in case of a failure.
func Bool(s string) bool {
	b, _ := strconv.ParseBool(strings.TrimSpace(s))

	return b
}

// IsTest states whether name is a valid test name (or other type, according to prefix).
//
// It checks if the next character after prefix is uppercase.
//
//	TestFoo    => true
//	Test       => true
//	TestfooBar => false
func IsTest(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}

	// "Test" is ok
	if len(name) == len(prefix) {
		return true
	}

	r, _ := utf8.DecodeRuneInString(name[len(prefix):])

	return !unicode.IsLower(r)
}
