package main

import (
	"github.com/alecthomas/kong"
	"github.com/emm035/procfly/internal/process"
)

type ProcflyFile struct {
	InlineTemplates map[string]string          `yaml:"templates"`
	TemplateFiles   map[string]string          `yaml:"template_files"`
	Procfile        map[string]process.Command `yaml:"procfile"`
	OnConfigChange  map[string]process.Command `yaml:"on_config_change"`
}

type Cli struct {
	Run     RunCmd     `name:"run" cmd:""`
	Version VersionCmd `name:"version" cmd:""`
}

func main() {
	ctx := kong.Parse(new(Cli))
	ctx.FatalIfErrorf(ctx.Run())
}
