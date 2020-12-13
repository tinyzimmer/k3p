package types

import (
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig"
)

func render(body []byte, vars map[string]string) ([]byte, error) {
	tmpl, err := template.New("").Funcs(sprig.TxtFuncMap()).Parse(string(body))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, map[string]interface{}{
		"Vars": vars,
	}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
