package cmdsuites

import "fmt"

type (
	Data struct {
		Package string
		Suite   Entity
		Test    Entity
		Tests   []Entity
	}

	Entity struct {
		Name string
		Pos  Pos
	}

	Pos struct {
		Dir      string
		Filename string
		Path     string
		Line     int
		Column   int
	}
)

func (e Entity) String() string {
	return e.Name
}

func (p Pos) String() string {
	return fmt.Sprintf("%s:%d:%d", p.Path, p.Line, p.Column)
}
