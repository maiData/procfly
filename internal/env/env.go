package env

import (
	"os"
	"strings"
)

type Env struct {
	Env map[string]string
	Fly FlyEnv
}

func Load() (env Env, err error) {
	env.Fly, err = loadFly()
	if err != nil {
		return
	}

	env.Env = make(map[string]string)
	for _, e := range os.Environ() {
		k, v, ok := strings.Cut(e, "=")
		if ok {
			env.Env[k] = v
		}
	}

	return
}
