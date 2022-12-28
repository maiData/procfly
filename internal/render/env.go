package render

import (
	"context"
	"os"
	"sort"
	"strings"

	"github.com/emm035/procfly/internal/file"
	"github.com/emm035/procfly/internal/privnet"
)

type Vars struct {
	Env     EnvVars
	Fly     FlyVars
	Procfly ProcflyVars
}

func LoadVars(paths file.Paths) (env Vars, err error) {
	env.Fly, err = loadFlyEnv()
	if err != nil {
		return
	}

	env.Env = loadEnv()

	env.Procfly = ProcflyVars{
		Root: paths.RootDir,
		File: paths.ProcflyFile,
	}
	return
}

type ProcflyVars struct {
	Root string
	File string
}

type EnvVars map[string]string

func loadEnv() EnvVars {
	env := make(EnvVars)
	for _, entry := range os.Environ() {
		// There may be some duplicates between these values
		// and those parsed into the fly config.
		if key, value, ok := strings.Cut(entry, "="); ok {
			env[key] = value
		}
	}
	return env
}

type FlyVars struct {
	Host           string
	AppName        string
	Region         string
	GatewayRegions []string
	IP             string
	Peers          []string
	ServerName     string
}

func loadFlyEnv() (env FlyVars, err error) {
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

	if ip, err := privnet.PrivateIPv6(); err != nil {
		return env, err
	} else {
		env.IP = ip.String()
	}

	if ips, err := privnet.AllPeers(
		context.Background(),
		env.AppName,
	); err != nil {
		return env, err
	} else {
		env.Peers = make([]string, len(ips))
		for i, ip := range ips {
			env.Peers[i] = ip.String()
		}
	}

	// easier to compare
	sort.Strings(env.GatewayRegions)
	return
}
