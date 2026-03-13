//go:build !windows

package rebuilder

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func terminateProcess(cmd *exec.Cmd) error {
	group, err := os.FindProcess(-cmd.Process.Pid)
	if err != nil {
		return err
	}

	return group.Signal(syscall.SIGTERM)
}

func notifyContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
}
