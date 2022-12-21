package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emm035/procfly/internal/file"
	"github.com/emm035/procfly/internal/process"
	"github.com/emm035/procfly/internal/render"
	"github.com/emm035/procfly/internal/util"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

type ProcflyFile struct {
	InlineTemplates map[string]string `yaml:"templates"`
	TemplateFiles   map[string]string `yaml:"template_files"`
	Processes       map[string]string `yaml:"procfile"`
	Reloaders       map[string]string `yaml:"reload"`
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

	err = validateTemplateNames(conf.TemplateFiles, conf.InlineTemplates)
	if err != nil {
		return err
	}

	err = validateReloaderNames(conf.Processes, conf.Reloaders)
	if err != nil {
		return err
	}

	e, err := render.LoadVars()
	if err != nil {
		return err
	}

	hash, err := renderTemplates(paths, conf, e)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, os.Kill, syscall.SIGTERM,
		syscall.SIGINT, syscall.SIGKILL)
	defer cancel()

	cmds, err := render.Commands("cmd_", conf.Processes, e)
	if err != nil {
		return err
	}

	svisor := process.NewSupervisor(ctx)
	for name, cmd := range cmds {
		svisor.RegisterProcess(name, cmd)
	}

	reloaders, err := render.Commands("reload_", conf.Reloaders, e)
	if err != nil {
		return err
	}
	for name, cmd := range reloaders {
		svisor.RegisterReload(name, cmd)
	}

	egrp, gctx := errgroup.WithContext(ctx)
	egrp.Go(svisor.Run)
	egrp.Go(watchEnv(gctx, svisor, hash, paths, conf))
	return egrp.Wait()
}

func watchEnv(ctx context.Context, svisor process.Supervisor, hash string, paths file.Paths, conf *ProcflyFile) func() error {
	return func() error {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		prev := hash

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				vars, err := render.LoadVars()
				if err != nil {
					return err
				}

				hash, err = renderTemplates(paths, conf, vars)
				if err != nil {
					return err
				}

				if hash == prev {
					continue
				} else {
					prev = hash
				}

				svisor.Log("procfly", "Running reloaders.")
				if err := svisor.Reload(); err != nil {
					return err
				}
			}
		}
	}
}

func renderTemplates(paths file.Paths, conf *ProcflyFile, vars render.Vars) (string, error) {
	inlinesHash, err := render.InlineTemplates(paths, conf.InlineTemplates, vars)
	if err != nil {
		return "", err
	}

	filesHash, err := render.TemplateFiles(paths, conf.TemplateFiles, vars)
	if err != nil {
		return "", err
	}

	return util.Hash(inlinesHash, filesHash)
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

func validateTemplateNames(a, b map[string]string) error {
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

func validateReloaderNames(proc, rld map[string]string) error {
	for k := range rld {
		if _, ok := proc[k]; !ok {
			return fmt.Errorf("reload: unknown proc: %s", k)
		}
	}
	return nil
}
