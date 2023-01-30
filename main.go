package main

import (
	"github.com/alecthomas/kong"
	"github.com/maidata/procfly/internal/cli"
)

type Cli struct {
	Run     cli.RunCmd     `name:"run" cmd:""`
	Version cli.VersionCmd `name:"version" cmd:""`
}

func main() {
	ctx := kong.Parse(new(Cli))
	ctx.FatalIfErrorf(ctx.Run())
}
