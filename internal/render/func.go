package render

import (
	"context"
	"fmt"
	"net"
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

	vars.VmAddrs = make([]string, len(vars.AllocIDs))
	for idx, allocID := range vars.AllocIDs {
		vars.VmAddrs[idx] = fmt.Sprintf("%s.vm.%s.internal", allocID, app)
	}

	vars.IPs, err = privnet.AllPeerIPs(ctx, app)
	if err != nil {
		return
	}

	return
}

type AppVars struct {
	Name     string
	IPs      []net.IPAddr
	AllocIDs []string
	VmAddrs  []string
}
