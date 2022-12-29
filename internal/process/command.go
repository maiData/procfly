package process

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Command struct {
	Name string
	Args []string
}

func (c *Command) UnmarshalText(p []byte) error {
	var quot *rune
	fields := strings.FieldsFunc(string(p), func(r rune) bool {
		switch r {
		case '\'', '"':
			if quot == nil {
				quot = &r
			} else {
				quot = nil
			}
			return false
		default:
			return r == ' ' && quot == nil
		}
	})

	for i, f := range fields {
		if f[0] == f[len(f)-1] && (f[0] == '"' || f[0] == '\'') {
			fields[i] = f[1 : len(f)-1]
		}
	}

	switch len(fields) {
	default:
		c.Args = fields[1:]
		fallthrough
	case 1:
		c.Name = fields[0]
	case 0:
		c.Name = string(p)
	}

	return nil
}

func (c Command) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%s %s", c.Name, strings.Join(c.Args, " "))), nil
}

func (c Command) Exec() *exec.Cmd {
	return exec.Command(c.Name, c.Args...)
}

func (c Command) ExecContext(ctx context.Context) *exec.Cmd {
	return exec.CommandContext(ctx, c.Name, c.Args...)
}
