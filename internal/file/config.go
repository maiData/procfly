package file

type ConfigFile struct {
	Template        []TemplateConfig `hcl:"template,block"`
	InitCommands    []CommandConfig  `hcl:"init,block"`
	ProcessCommands []CommandConfig  `hcl:"process,block"`
	ReloadCommands  []CommandConfig  `hcl:"reload,block"`
}

type CommandConfig struct {
	Name       string `hcl:"name,label"`
	Command    string `hcl:"command"`
	KillSignal string `hcl:"kill_signal"`
}

type TemplateConfig struct {
	Path     string `hcl:"path,label"`
	Template string `hcl:"template"`
	File     string `hcl:"file"`
}
