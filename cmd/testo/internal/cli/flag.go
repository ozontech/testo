package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"text/template"
)

var _ flag.Value = (*FlagTemplate)(nil)

type FlagTemplate struct {
	template *template.Template
	source   string
}

func (f *FlagTemplate) Execute(v any) (string, error) {
	var buf bytes.Buffer

	if err := f.template.Execute(&buf, v); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (f *FlagTemplate) Set(s string) error {
	funcs := template.FuncMap{
		"json": func(v any) string {
			m, err := json.Marshal(v)
			if err != nil {
				return err.Error()
			}

			return string(m)
		},
	}

	parsed, err := template.New("flag").Funcs(funcs).Parse(s)
	if err != nil {
		return err
	}

	f.template = parsed
	f.source = s

	return nil
}

func (f *FlagTemplate) String() string {
	return f.source
}
