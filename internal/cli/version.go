package cli

import (
	"errors"
	"fmt"
	"runtime/debug"
)

type VersionCmd struct {
	Debug bool `name:"debug" short:"d"`
}

func (cmd *VersionCmd) Run() error {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return errors.New("unable to read build information")
	}
	if cmd.Debug {
		fmt.Printf("%+v\n", bi)
	} else {
		fmt.Println(bi.Main.Version)
	}
	return nil
}
