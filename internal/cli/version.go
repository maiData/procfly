package cli

import (
	"fmt"

	"github.com/emm035/gravel/pkg/buildinfo"
)

type VersionCmd struct {
}

func (cmd *VersionCmd) Run() error {
	fmt.Println(buildinfo.GetVersion())
	return nil
}
