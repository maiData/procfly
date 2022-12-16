package main

import (
	"github.com/alecthomas/kong"
)

type Cli struct {
	Run     RunCmd     `name:"run" cmd:""`
	Version VersionCmd `name:"version" cmd:""`
}

func main() {
	ctx := kong.Parse(new(Cli))
	ctx.FatalIfErrorf(ctx.Run())
}
