package render

import (
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/emm035/procfly/internal/privnet"
)

var funcs = template.FuncMap{
	"timestamp": time.Now,
	"lookup":    lookupApp,
}

func lookupApp(app string) (vars AppVars, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	vars.Name = app
	vars.AllocIDs, err = privnet.AllPeerAllocIDs(ctx, app)
	if err != nil {
		return
	}

	vars.VMAddrs = make([]string, len(vars.AllocIDs))
	for idx, allocID := range vars.AllocIDs {
		vars.VMAddrs[idx] = fmt.Sprintf("%s.vm.%s.internal", allocID, app)
	}

	return
}

type AppVars struct {
	Name     string
	AllocIDs []string
	VMAddrs  []string
}
