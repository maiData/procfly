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

func Fly() (FlyEnv, error) {
	host := "fly-local-6pn"
	appName := os.Getenv("FLY_APP_NAME")

	var regions []string
	var err error

	if appName != "" {
		regions, err = privnet.GetRegions(context.Background(), appName)
	} else {
		// defaults for local exec
		host = "localhost"
		appName = "local"
		regions = []string{"local"}
	}

	// easier to compare
	sort.Strings(regions)

	region := os.Getenv("FLY_REGION")
	if region == "" {
		region = "local"
	}

	vars := FlyEnv{
		AppName:        appName,
		Region:         region,
		GatewayRegions: regions,
		Host:           host,
		ServerName:     os.Getenv("FLY_ALLOC_ID"),
		Timestamp:      time.Now(),
	}
	if err != nil {
		return FlyEnv{}, err
	}
	return vars, nil
}
