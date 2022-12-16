package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/emm035/procfly/internal/env"
	"github.com/emm035/procfly/internal/file"
	"github.com/emm035/procfly/internal/process"
	"github.com/emm035/procfly/internal/render"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

type ProcflyFile struct {
	InlineTemplates map[string]string          `yaml:"templates"`
	TemplateFiles   map[string]string          `yaml:"template_files"`
	Procfile        map[string]process.Command `yaml:"procfile"`
	OnConfigChange  map[string]process.Command `yaml:"reload"`
}

type RunCmd struct {
	ProcflyDir string `arg:"" name:"procfly-dir" type:"existingFile" default:"."`
}

func (cli *RunCmd) Run() error {
	paths := file.NewPaths(cli.ProcflyDir)

	conf, err := loadProcflyFile(paths.ProcflyFile)
	if err != nil {
		return err
	}

	err = assertDistinctTemplates(conf.TemplateFiles, conf.InlineTemplates)
	if err != nil {
		return err
	}

	err = assertValidChangeActions(conf.Procfile, conf.OnConfigChange)
	if err != nil {
		return err
	}

	e, err := env.Load()
	if err != nil {
		return err
	}

	err = renderTemplates(paths, conf, e)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, os.Kill, syscall.SIGTERM,
		syscall.SIGINT, syscall.SIGKILL)
	defer cancel()

	svisor := process.NewSupervisor(ctx, conf.Procfile, conf.OnConfigChange)

	egrp, gctx := errgroup.WithContext(ctx)
	egrp.Go(svisor.Run)
	egrp.Go(func() error {
		return watchEnv(gctx, svisor, e, paths, conf)
	})
	return egrp.Wait()
}

func watchEnv(ctx context.Context, svisor process.Supervisor, first env.Env, paths file.Paths, conf *ProcflyFile) error {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	latest := first

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			e, err := env.Load()
			if err != nil {
				return err
			}

			if reflect.DeepEqual(e, latest) {
				continue
			}

			err = renderTemplates(paths, conf, e)
			if err != nil {
				return err
			}

			if err := svisor.Reload(); err != nil {
				return err
			}
		}
	}
}

func renderTemplates(paths file.Paths, conf *ProcflyFile, e env.Env) error {
	err := render.InlineTemplates(paths, conf.InlineTemplates, e)
	if err != nil {
		return err
	}

	err = render.TemplateFiles(paths, conf.TemplateFiles, e)
	if err != nil {
		return err
	}

	return nil
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

func assertDistinctTemplates(a, b map[string]string) error {
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

func assertValidChangeActions(proc, rld map[string]process.Command) error {
	for k := range rld {
		if _, ok := proc[k]; !ok {
			return fmt.Errorf("reload: unknown proc: %s", k)
		}
	}
	return nil
}
