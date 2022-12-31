package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emm035/procfly/internal/file"
	"github.com/emm035/procfly/internal/process"
	"github.com/emm035/procfly/internal/render"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

type ProcflyFile struct {
	InlineTemplates map[string]string `yaml:"templates"`
	TemplateFiles   map[string]string `yaml:"template_files"`
	Init            map[string]string `yaml:"init"`
	Processes       map[string]string `yaml:"processes"`
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

	vars, err := render.LoadVars(paths)
	if err != nil {
		return err
	}

	rndr := render.NewRenderer(paths, vars)

	if err := renderTemplatedFiles(rndr, conf); err != nil {
		return err
	}

	inits, err := rndr.Commands(conf.Init)
	if err != nil {
		return err
	}

	cmds, err := rndr.Commands(conf.Processes)
	if err != nil {
		return err
	}

	reloaders, err := rndr.Commands(conf.Reloaders)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, os.Kill, syscall.SIGTERM,
		syscall.SIGINT, syscall.SIGKILL)
	defer cancel()
	egrp, gctx := errgroup.WithContext(ctx)

	// Create a process supervisor, registering
	// all process & reload commands.
	svisor := process.NewSupervisor(gctx)
	for name, cmd := range inits {
		svisor.RegisterInit(name, cmd)
	}
	for name, cmd := range cmds {
		svisor.RegisterProcess(name, cmd)
	}
	for name, cmd := range reloaders {
		svisor.RegisterReload(name, cmd)
	}

	// Run the supervisor and environment watcher.
	// If either exits with an error, gctx will be
	// cancelled, and the other should stop.
	egrp.Go(svisor.Run)
	egrp.Go(watchEnv(gctx, svisor, paths, rndr, conf))

	// Wait for something to fail out, or for a
	// signal to be received, telling us to exit.
	return egrp.Wait()
}

func watchEnv(ctx context.Context, svisor process.Supervisor, paths file.Paths, renderer *render.Renderer, conf *ProcflyFile) func() error {
	return func() error {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		prev := renderer.Hash()

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				// We should periodically reload the rendering variables,
				// and reset the renderer so its hash will be reset. This
				// lets us figure out whether any configurations have
				// been changed by an update to the vars
				if vars, err := render.LoadVars(paths); err != nil {
					return err
				} else {
					renderer.Reset(vars)
				}

				if err := renderTemplatedFiles(renderer, conf); err != nil {
					return err
				}

				// If the hash of our templated files hasn't changed,
				// we should skip running our reloaders.
				if hash := renderer.Hash(); hash == prev {
					continue
				} else {
					prev = hash
				}

				svisor.Log("procfly", "Running reloaders.")
				if err := svisor.Reload(); err == nil || errors.Is(err, process.ErrNotRunning) {
					continue
				} else {
					return err
				}
			}
		}
	}
}

func renderTemplatedFiles(renderer *render.Renderer, conf *ProcflyFile) error {
	if err := renderer.InlineTemplates(conf.InlineTemplates); err != nil {
		return err
	}

	if err := renderer.TemplateFiles(conf.TemplateFiles); err != nil {
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
