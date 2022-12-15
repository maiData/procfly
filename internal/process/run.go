package process

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/lipgloss"
	"github.com/emm035/procfly/internal/file"
	"golang.org/x/sync/errgroup"
)

// https://observablehq.com/@d3/color-schemes#Category10
var colors = []lipgloss.Style{
	lipgloss.NewStyle().Foreground(lipgloss.Color("#1f77b4")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7f0e")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#2ca02c")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#d62728")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#9467bd")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#8c564b")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#e377c2")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#7f7f7f")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#bcbd22")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#17becf")),
}

func Run(ctx context.Context, paths file.Paths, cmds map[string]Command) error {
	egrp, gctx := errgroup.WithContext(ctx)

	maxlen := 0
	for name := range cmds {
		if len(name) > maxlen {
			maxlen = len(name)
		}
	}

	pfw := NewPrefixWriterFactory(os.Stdout)

	i := 0
	for name, cmd := range cmds {
		_name := name
		prefix := fmt.Sprintf(fmt.Sprintf("%%-%ds | ", maxlen), name)
		prefix = colors[i%len(cmds)].Render(prefix)
		i++

		c := exec.CommandContext(gctx, cmd.Name, cmd.Args...)
		c.Dir = paths.RootDir
		c.Stderr = pfw.Writer(prefix)
		c.Stdout = pfw.Writer(prefix)

		egrp.Go(func() error {
			if err := c.Run(); err != nil {
				if errors.Is(ctx.Err(), context.Canceled) {
					return nil
				}
				return fmt.Errorf("%s: %w", _name, err)
			}
			return nil
		})
	}

	return egrp.Wait()
}
