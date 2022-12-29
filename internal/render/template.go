package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"os"
	"text/template"
	"time"

	"github.com/emm035/procfly/internal/file"
	"github.com/emm035/procfly/internal/process"
	"github.com/emm035/procfly/internal/util"
)

var templates = make(map[string]*template.Template)

var funcs = template.FuncMap{
	"timestamp": time.Now,
}

type Renderer struct {
	paths file.Paths
	vars  any
	hash  hash.Hash
}

func NewRenderer(paths file.Paths, vars any) *Renderer {
	return &Renderer{
		paths: paths,
		vars:  vars,
		hash:  sha256.New(),
	}
}

func (r *Renderer) Hash() string {
	return hex.EncodeToString(r.hash.Sum(nil))
}

func (r *Renderer) Reset(vars any) {
	r.hash = sha256.New()
	if vars != nil {
		r.vars = vars
	}
}

func (r *Renderer) Commands(tmpls map[string]string) (map[string]process.Command, error) {
	rendered := make(map[string]process.Command)
	buf := new(bytes.Buffer)
	cmd := new(process.Command)
	for name, tmpl := range tmpls {
		if err := r.render(tmpl, false, buf); err != nil {
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

func (r *Renderer) InlineTemplates(tmpls map[string]string) error {
	for file, tmpl := range tmpls {
		f, err := r.paths.Open(file, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0660)
		if err != nil {
			return err
		}

		if err := r.render(tmpl, true, f); err != nil {
			return err
		}
	}
	return nil
}

func (r *Renderer) TemplateFiles(tmpls map[string]string) error {
	for file, tmplf := range tmpls {
		tmpl, err := r.paths.Read(tmplf)
		if err != nil {
			return err
		}

		f, err := r.paths.Open(file, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}

		if err := r.render(string(tmpl), true, f); err != nil {
			return err
		}
	}
	return nil
}

func (r *Renderer) render(tmpl string, hash bool, file io.Writer) (err error) {
	// Deduplicate templates by hashing them
	name, err := util.Hash(tmpl)
	if err != nil {
		return err
	}

	t, ok := templates[name]
	if !ok {
		templates[name], err = template.New(name).Funcs(funcs).Parse(tmpl)
		if err != nil {
			return err
		}
		t = templates[name]
	}

	var w io.Writer = file
	if hash {
		w = io.MultiWriter(r.hash, file)
	}

	if err := t.Execute(w, r.vars); err != nil {
		return err
	}

	return nil
}
