package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/emm035/procfly/internal/env"
	"github.com/emm035/procfly/internal/file"
	"github.com/emm035/procfly/internal/process"
	"github.com/emm035/procfly/internal/render"
	"gopkg.in/yaml.v3"
)

type RunCmd struct {
	ProcflyDir string `arg:"" name:"procfly-dir" type:"existingFile" default:"."`
}

func (cli *RunCmd) Run() error {
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