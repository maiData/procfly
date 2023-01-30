package cli

import (
	"errors"
	"fmt"
	"runtime/debug"
)

type VersionCmd struct {
}

func (cmd *VersionCmd) Run() error {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return errors.New("unable to read build information")
	}
	fmt.Printf("%+v\n", bi)
	return nil
}
