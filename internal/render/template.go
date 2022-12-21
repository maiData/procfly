package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"html/template"
	"io"
	"os"
	"time"

	"github.com/emm035/procfly/internal/file"
	"github.com/emm035/procfly/internal/process"
)

var templates = make(map[string]*template.Template)

var funcs = template.FuncMap{
	"timestamp": time.Now,
}

func Commands(prefix string, tmpls map[string]string, values any) (map[string]process.Command, error) {
	rendered := make(map[string]process.Command)
	buf := new(bytes.Buffer)
	cmd := new(process.Command)
	for name, tmpl := range tmpls {
		if err := render(prefix+name, tmpl, values, nil, buf); err != nil {
			return nil, err
		}

		if err := cmd.UnmarshalText(buf.Bytes()); err != nil {
			return nil, err
		}
		rendered[name] = *cmd
		buf.Reset()
	}
	return rendered, nil
}

func InlineTemplates(paths file.Paths, tmpls map[string]string, values any) (string, error) {
	hash := sha256.New()
	for file, tmpl := range tmpls {
		f, err := openFile(paths, file)
		if err != nil {
			return "", err
		}

		if err := render(file, tmpl, values, hash, f); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func TemplateFiles(paths file.Paths, tmpls map[string]string, vars any) (string, error) {
	hash := sha256.New()
	for file, tmplf := range tmpls {
		tmpl, err := os.ReadFile(paths.Rel(tmplf))
		if err != nil {
			return "", err
		}

		f, err := os.OpenFile(paths.Rel(file), os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return "", err
		}

		if err := render(file, string(tmpl), vars, hash, f); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func openFile(paths file.Paths, file string) (*os.File, error) {
	return os.OpenFile(paths.Rel(file), os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
}

func render(name, tmpl string, vars any, hash, file io.Writer) (err error) {
	t, ok := templates[name]
	if !ok {
		templates[name], err = template.New(name).Funcs(funcs).Parse(tmpl)
		if err != nil {
			return err
		}
		t = templates[name]
	}

	var w io.Writer = file
	if hash != nil {
		w = io.MultiWriter(hash, file)
	}

	if err := t.Execute(w, vars); err != nil {
		return err
	}

	return nil
}
