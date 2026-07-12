package cmdsuites

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"unicode"
)

//go:embed template.go
var currentFile string

//types:start
type Data struct {
	Package Package  // suite package
	Suite   Entity   // current suite
	Test    Entity   // current test for this suite
	Tests   []Entity // all tests
}

type Package struct {
	Name string
	Path string
	Dir  string
}

type Entity struct {
	Name string
	Pos  Pos
}

type Pos struct {
	Dir      string // file dir
	Filename string // base file name
	Path     string // absolute path
	Line     int    // 1-based line number
	Column   int    // 1-based column number
}

//types:end

func (p Package) String() string {
	return p.Name
}

func (e Entity) String() string {
	return e.Name
}

func (p Pos) String() string {
	return fmt.Sprintf("%s:%d:%d", p.Path, p.Line, p.Column)
}

func templateTypes(indent string) string {
	const start = "//types:start"
	const end = "//types:end"

	var inTypes bool

	var buf bytes.Buffer

lines:
	for line := range strings.Lines(currentFile) {
		switch {
		case strings.HasPrefix(line, start):
			inTypes = true

		case strings.HasPrefix(line, end):
			break lines

		case inTypes:
			buf.WriteString(indent)
			buf.WriteString(strings.TrimRightFunc(line, unicode.IsSpace))
			buf.WriteString("\n")
		}
	}

	return strings.TrimRightFunc(buf.String(), unicode.IsSpace)
}
