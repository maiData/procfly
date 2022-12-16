package process

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sync/errgroup"
)

var (
	ErrRunning    = errors.New("supervisor is running")
	ErrNotRunning = errors.New("supervisor is not running")
)

type OnChange struct {
	Run Command `yaml:"run"`
}

type Supervisor interface {
	RegisterProcess(string, Command)
	RegisterReload(string, Command)
	// Run all of the supervisor's registered
	// commands
	Run() error
	// Run all of the supervisor's registered
	// reload scripts
	Reload() error
	// Log a message with the given prefix, using
	// the supervisor's multiplexed (prefixed) writer
	Log(name, message string)
}

type svisor struct {
	root context.Context
	sout MuxWriter
	lock sync.Mutex
	ctxs map[string]context.Context
	cmds map[string]Command
	rlds map[string]Command
	proc map[string]*exec.Cmd
}

func NewSupervisor(ctx context.Context) Supervisor {
	return &svisor{
		root: ctx,
		sout: NewMuxWriter(os.Stdout),
		ctxs: make(map[string]context.Context),
		cmds: make(map[string]Command),
		rlds: make(map[string]Command),
		proc: make(map[string]*exec.Cmd),
	}
}

func (sv *svisor) RegisterProcess(name string, cmd Command) {
	// We can pre-register known names to reduce the
	// chances of the log prefix being resized during
	// execution of the processes.
	sv.sout.RegisterName(name)
	sv.cmds[name] = cmd
}

func (sv *svisor) RegisterReload(name string, cmd Command) {
	// We can pre-register known names to reduce the
	// chances of the log prefix being resized during
	// execution of the processes.
	sv.sout.RegisterName("reload_" + name)
	sv.rlds[name] = cmd
}

func (sv *svisor) Run() error {
	if !sv.lock.TryLock() {
		// If we can't acquire the lock, that means
		// this supervisor is already running.
		return ErrRunning
	} else {
		defer sv.lock.Unlock()
	}

	egrp, gctx := errgroup.WithContext(sv.root)
	for name, cmd := range sv.cmds {
		proc := cmd.Exec()
		if err := sv.run(name, proc); err != nil {
			return err
		}

		sv.proc[name] = proc
		defer delete(sv.proc, name)

		egrp.Go(func() error {
			return sv.stop(gctx, proc)
		})
	}

	if err := egrp.Wait(); !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func (sv *svisor) run(name string, cmd *exec.Cmd) error {
	pseu, term, err := pty.Open()
	if err != nil {
		return err
	}

	// We need to copy output from the pseudo-terminal
	// over to stdout
	go io.Copy(sv.sout.Writer(name), pseu)

	// Set file descriptors on process
	cmd.Stdout = term
	cmd.Stderr = term
	cmd.Stdin = term
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
	}

	return cmd.Start()
}

func (sv *svisor) stop(ctx context.Context, cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return ErrNotRunning
	}

	errc := make(chan error)
	go func() {
		select {
		case errc <- nil:
			return
		case <-ctx.Done():
		}

		err := cmd.Process.Signal(os.Interrupt)
		if errors.Is(err, os.ErrProcessDone) {
			errc <- nil
			return
		}

		timer := time.NewTimer(5 * time.Second)
		select {
		// Report ctx.Err() as the reason we interrupted the process...
		case errc <- ctx.Err():
			timer.Stop()
			return
		// ...but after killDelay has elapsed, fall back to a stronger signal.
		case <-timer.C:
		}

		// Wait still hasn't returned.
		// Kill the process harder to make sure that it exits.
		//
		// Ignore any error: if cmd.Process has already terminated, we still
		// want to send ctx.Err() (or the error from the Interrupt call)
		// to properly attribute the signal that may have terminated it.
		_ = cmd.Process.Kill()
		errc <- err
	}()

	waitErr := cmd.Wait()
	if interruptErr := <-errc; interruptErr != nil {
		return interruptErr
	}
	return waitErr
}

func (sv *svisor) Reload() error {
	if sv.lock.TryLock() {
		defer sv.lock.Unlock()
		return ErrNotRunning
	}

	ctx, cancel := context.WithTimeout(sv.root, 5*time.Second)
	defer cancel()

	egrp, gctx := errgroup.WithContext(ctx)

	for name, cmd := range sv.rlds {
		proc := cmd.Exec()
		if err := sv.run("reload_"+name, proc); err != nil {
			cancel()
			return err
		}

		egrp.Go(func() error {
			return sv.stop(gctx, proc)
		})
	}

	err := egrp.Wait()
	if errors.Is(err, context.Canceled) {
		// Ignore shutdown due to cancellation, because
		// that means an external signal caused the interruption
		return nil
	}
	return err
}

func (sv *svisor) Log(name, message string) {
	fmt.Fprintln(sv.sout.Writer(name), message)
}
