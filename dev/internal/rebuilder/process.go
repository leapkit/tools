package rebuilder

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func newProcess(e entry) *process {
	return &process{
		entry:  e,
		Stdout: wrap(os.Stdout, e),
		Stderr: wrap(os.Stderr, e),
	}
}

type process struct {
	entry
	Stdout io.Writer
	Stderr io.Writer
}

func (p *process) Run(parentCtx context.Context, reload chan bool) error {
	fields := strings.Fields(p.Command)
	if len(fields) == 0 {
		return fmt.Errorf("empty command for process %q", p.Name)
	}

	name, args := fields[0], fields[1:]

	var restarted bool

	for {
		ctx, cancel := context.WithCancel(context.Background())

		cmd := exec.CommandContext(ctx, name, args...)

		cmd.Stdout = p.Stdout
		cmd.Stderr = p.Stderr
		setSysProcAttr(cmd)

		if restarted {
			fmt.Fprintln(p.Stdout, "Restarted...")
		}

		if err := cmd.Start(); err != nil {
			cancel()
			fmt.Fprintf(p.Stderr, "failed to start process: %v\n", err)
			return err
		}

		errCh := make(chan error, 1)
		go func() {
			errCh <- cmd.Wait()
		}()

		select {
		case <-reload:
			if err := terminateProcess(cmd); err != nil {
				fmt.Fprintf(p.Stdout, "error restarting process: %v\n", err)
			}
			<-errCh
		case <-parentCtx.Done():
			fmt.Fprintln(p.Stdout, "Stopping...")
			if err := terminateProcess(cmd); err != nil {
				fmt.Fprintf(p.Stdout, "error stopping process: %v\n", err)
			}
			<-errCh

			cancel()
			return nil
		case err := <-errCh:
			if err != nil {
				fmt.Fprintf(p.Stderr, "process exited with error: %v\n", err)
			}

			select {
			case <-reload:
			case <-parentCtx.Done():
				cancel()
				return nil
			}
		}

		cancel()
		restarted = true
	}
}
