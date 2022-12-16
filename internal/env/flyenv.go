package env

import (
	"context"
	"os"
	"sort"
	"time"

	"github.com/emm035/procfly/internal/privnet"
)

type FlyEnv struct {
	Host           string
	AppName        string
	Region         string
	GatewayRegions []string
	ServerName     string
	Timestamp      time.Time
}

func loadFly() (env FlyEnv, err error) {
	env.Timestamp = time.Now()

	env.ServerName = os.Getenv("FLY_ALLOC_ID")
	if env.ServerName == "" {
		env.ServerName = "local"
	}

	env.Region = os.Getenv("FLY_REGION")
	if env.Region == "" {
		env.Region = "local"
	}

	env.AppName = os.Getenv("FLY_APP_NAME")
	if env.AppName == "" {
		env.Host = "localhost"
		env.AppName = "local"
		env.GatewayRegions = []string{"local"}
		return
	}

	env.Host = "fly-local-6pn"
	if env.GatewayRegions, err = privnet.GetRegions(
		context.Background(),
		env.AppName,
	); err != nil {
		return
	}

	// easier to compare
	sort.Strings(env.GatewayRegions)
	return
}
