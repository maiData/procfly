package render

import (
	"html/template"
	"os"

	"github.com/emm035/procfly/internal/file"
)

func InlineTemplates(paths file.Paths, tmpls map[string]string, values any) error {
	for file, tmpl := range tmpls {
		if err := renderFile(paths.Rel(file), tmpl, values); err != nil {
			return err
		}
	}
	return nil
}

func TemplateFiles(paths file.Paths, tmpls map[string]string, values any) error {
	for file, tmplf := range tmpls {
		tmpl, err := os.ReadFile(paths.Rel(tmplf))
		if err != nil {
			return err
		}

		if err := renderFile(paths.Rel(file), string(tmpl), values); err != nil {
			return err
		}
	}
	return nil
}

func renderFile(file, tmpl string, values any) error {
	t, err := template.New(file).Parse(tmpl)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(file, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	if err := t.Execute(f, values); err != nil {
		return err
	}

	return nil
}
