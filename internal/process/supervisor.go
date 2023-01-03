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

	"github.com/cenkalti/backoff/v4"
	"github.com/creack/pty"
	"golang.org/x/sync/errgroup"
)

var (
	ErrAlreadyRunning  = errors.New("supervisor is running")
	ErrNotRunning      = errors.New("supervisor is not running")
	ErrExitedWithCode  = errors.New("exited with error code")
	ErrExitedWithError = errors.New("exited with error")
)

type OnChange struct {
	Run Command `yaml:"run"`
}

type Supervisor interface {
	RegisterInit(string, Command)
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
	Logf(name, message string, args ...any)
}

type supervisor struct {
	root  context.Context
	sout  MuxWriter
	lock  sync.Mutex
	ctxs  map[string]context.Context
	inits map[string]Command
	cmds  map[string]Command
	rlds  map[string]Command
}

func NewSupervisor(ctx context.Context) Supervisor {
	return &supervisor{
		root:  ctx,
		sout:  NewMuxWriter(os.Stdout),
		ctxs:  make(map[string]context.Context),
		inits: make(map[string]Command),
		cmds:  make(map[string]Command),
		rlds:  make(map[string]Command),
	}
}

func (sv *supervisor) RegisterInit(name string, cmd Command) {
	// We can pre-register known names to reduce the
	// chances of the log prefix being resized during
	// execution of the processes.
	sv.sout.RegisterName("init_" + name)
	sv.inits[name] = cmd
}

func (sv *supervisor) RegisterProcess(name string, cmd Command) {
	// We can pre-register known names to reduce the
	// chances of the log prefix being resized during
	// execution of the processes.
	sv.sout.RegisterName(name)
	sv.cmds[name] = cmd
}

func (sv *supervisor) RegisterReload(name string, cmd Command) {
	// We can pre-register known names to reduce the
	// chances of the log prefix being resized during
	// execution of the processes.
	sv.sout.RegisterName("reload_" + name)
	sv.rlds[name] = cmd
}

func (sv *supervisor) Run() error {
	if !sv.lock.TryLock() {
		// If we can't acquire the lock, that means
		// this supervisor is already running.
		return ErrAlreadyRunning
	} else {
		defer sv.lock.Unlock()
	}

	if len(sv.inits) > 0 {
		sv.Log("procfly", "Running initializers.")
		if err := sv.runInits(); err != nil {
			return err
		}
		sv.Log("procfly", "Initializers complete.")
	}

	return sv.runProcesses()
}

func (sv *supervisor) runInits() error {
	// Init commands should take < 10s
	ctx, cancel := context.WithTimeout(sv.root, 10*time.Second)
	defer cancel()

	egrp, gctx := errgroup.WithContext(ctx)
	for name, cmd := range sv.inits {
		_cmd, _name := cmd, name
		egrp.Go(sv.run(gctx, "init_"+_name, _cmd))
	}
	return egrp.Wait()
}

func (sv *supervisor) runProcesses() error {
	ctx, cancel := context.WithCancel(sv.root)
	defer cancel()

	egrp, gctx := errgroup.WithContext(ctx)
	for name, cmd := range sv.cmds {
		_cmd, _name := cmd, name
		egrp.Go(sv.withRestarts(gctx, _name, sv.run(gctx, _name, _cmd)))
	}

	if err := egrp.Wait(); !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func (sv *supervisor) setupStdout(name string, cmd *exec.Cmd) error {
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

	return nil
}

func (sv *supervisor) withRestarts(ctx context.Context, name string, fn func() error) func() error {
	boff := backoff.NewExponentialBackOff()
	boff.MaxInterval = 15 * time.Second
	boff.Multiplier = 2
	boff.InitialInterval = 1 * time.Second

	return func() error {
		for {
			if err := fn(); errors.Is(err, ErrExitedWithCode) || errors.Is(err, ErrExitedWithError) {
				nboff := boff.NextBackOff()
				sv.Logf(name, err.Error())
				sv.Logf("procfly", "Waiting %s before restarting %s", nboff, name)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(nboff):
					continue
				}
			} else {
				return nil
			}
		}
	}
}

func (sv *supervisor) run(ctx context.Context, name string, command Command) func() error {
	return func() error {
		sv.Logf("procfly", "Start %s: %s", name, command)
		cmd := command.Exec()

		if err := sv.setupStdout(name, cmd); err != nil {
			return err
		}

		if err := cmd.Start(); err != nil {
			return err
		}

		sch := make(chan *os.ProcessState)

		go func() {
			// Wait for the process to exit. Once it has, we can
			// cancel the context that's waiting for it.
			state, err := cmd.Process.Wait()
			if err != nil {
				close(sch)
			} else {
				sch <- state
				close(sch)
			}
		}()

		select {
		case <-ctx.Done():
			// We need to kill the process gracefully. Send an
			// interrupt signal telling it to shut down.
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				return err
			}
		case state, ok := <-sch:
			// The process exited before we told it to. If it
			// had a non-zero exit code, we should return an error
			// stating that.
			if !ok {
				return ErrExitedWithError
			} else if state.ExitCode() != 0 {
				return fmt.Errorf("%s: %w %d", name, ErrExitedWithCode, cmd.ProcessState.ExitCode())
			} else {
				return nil
			}
		}

		select {
		case <-time.After(5 * time.Second):
			// The process is still running after 5 seconds, so
			// we need to kill it more forcefully. If we're in this
			// branch, that means the process is being killed, so we
			// should ignore the error message from killing, and
			// return the one from the context that caused it.
			_ = cmd.Process.Kill()
			return ctx.Err()
		case <-sch:
			// Sending the signal managed to shut down the process
			// gracefully. We can exit without an error.
			return nil
		}
	}
}

func (sv *supervisor) Reload() error {
	if sv.lock.TryLock() {
		defer sv.lock.Unlock()
		return ErrNotRunning
	}

	ctx, cancel := context.WithTimeout(sv.root, 5*time.Second)
	defer cancel()

	egrp, gctx := errgroup.WithContext(ctx)

	for name, cmd := range sv.rlds {
		_cmd, _name := cmd, name
		egrp.Go(sv.run(gctx, "reload_"+_name, _cmd))
	}

	err := egrp.Wait()
	if errors.Is(err, context.Canceled) {
		// Ignore shutdown due to cancellation, because
		// that means an external signal caused the interruption
		return nil
	}
	return err
}

func (sv *supervisor) Log(name, message string) {
	fmt.Fprintln(sv.sout.Writer(name), message)
}

func (sv *supervisor) Logf(name, message string, args ...any) {
	fmt.Fprintf(sv.sout.Writer(name), message, args...)
}
