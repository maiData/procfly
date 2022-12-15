package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/emm035/procfly/internal/env"
	"github.com/emm035/procfly/internal/file"
	"github.com/emm035/procfly/internal/process"
	"github.com/emm035/procfly/internal/render"
	"gopkg.in/yaml.v3"
)

type ProcflyFile struct {
	InlineTemplates map[string]string          `yaml:"templates"`
	TemplateFiles   map[string]string          `yaml:"template_files"`
	Procfile        map[string]process.Command `yaml:"procfile"`
	OnConfigChange  map[string]process.Command `yaml:"on_config_change"`
}

func main() {
	ctx := kong.Parse(new(Cli))
	ctx.FatalIfErrorf(ctx.Run())
}

type Cli struct {
	ProcflyDir string `arg:"" name:"procfly-dir" type:"existingFile" default:"."`
}

func (cli *Cli) Run() error {
	paths := file.NewPaths(cli.ProcflyDir)

	conf, err := loadProcflyFile(paths.ProcflyFile)
	if err != nil {
		return err
	}

	err = assertDistinctKeys(conf.TemplateFiles, conf.InlineTemplates)
	if err != nil {
		return err
	}

	flyenv, err := env.Fly()
	if err != nil {
		return err
	}

	err = render.InlineTemplates(paths, conf.InlineTemplates, flyenv)
	if err != nil {
		return err
	}

	err = render.TemplateFiles(paths, conf.TemplateFiles, flyenv)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, os.Kill, syscall.SIGTERM,
		syscall.SIGINT, syscall.SIGKILL)
	defer cancel()

	return process.Run(ctx, paths, conf.Procfile)
}

func loadProcflyFile(file string) (*ProcflyFile, error) {
	pfile, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer pfile.Close()

	conf := new(ProcflyFile)
	err = yaml.NewDecoder(pfile).Decode(conf)
	if err != nil {
		return nil, err
	}
	return conf, err
}

func assertDistinctKeys(a, b map[string]string) error {
	for k := range a {
		if _, ok := b[k]; ok {
			return fmt.Errorf("multiple templates: %s", k)
		}
	}
	for k := range b {
		if _, ok := a[k]; ok {
			return fmt.Errorf("multiple templates: %s", k)
		}
	}
	return nil
}
