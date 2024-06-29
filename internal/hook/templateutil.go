package hook

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

func RenderTemplate(text string, vars interface{}) (string, error) {
	template, err := template.New("template").Parse(text)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	buf := &bytes.Buffer{}
	if err := template.Execute(buf, vars); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

func RenderTemplateOrDefault(template string, defaultTmpl string, vars interface{}) (string, error) {
	if strings.Trim(template, " ") == "" {
		return RenderTemplate(defaultTmpl, vars)
	}
	return RenderTemplate(template, vars)
}
