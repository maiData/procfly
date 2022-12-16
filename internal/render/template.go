package render

import (
	"crypto/sha256"
	"encoding/hex"
	"html/template"
	"io"
	"os"
	"time"

	"github.com/emm035/procfly/internal/file"
)

var funcs = template.FuncMap{
	"timestamp": time.Now,
}

func InlineTemplates(paths file.Paths, tmpls map[string]string, values any) (string, error) {
	hash := sha256.New()
	for file, tmpl := range tmpls {
		if err := renderFile(hash, paths.Rel(file), tmpl, values); err != nil {
			return "", nil
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func TemplateFiles(paths file.Paths, tmpls map[string]string, values any) (string, error) {
	hash := sha256.New()
	for file, tmplf := range tmpls {
		tmpl, err := os.ReadFile(paths.Rel(tmplf))
		if err != nil {
			return "", err
		}

		if err := renderFile(hash, paths.Rel(file), string(tmpl), values); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func renderFile(hash io.Writer, file, tmpl string, values any) error {
	t, err := template.New(file).Funcs(funcs).Parse(tmpl)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(file, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	if err := t.Execute(io.MultiWriter(hash, f), values); err != nil {
		return err
	}

	return nil
}
