package email

import (
	"bytes"
	"path/filepath"
	"text/template"
)

type TextTemplateLoader struct {
	basePath string
}

func NewTextTemplateLoader(basePath string) *TextTemplateLoader {
	return &TextTemplateLoader{basePath: basePath}
}

func (f *TextTemplateLoader) Render(name string, data any) (string, error) {
	templatePath := filepath.Join(f.basePath, name)
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
