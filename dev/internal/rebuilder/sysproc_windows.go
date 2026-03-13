//go:build windows

package rebuilder

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
)

func setSysProcAttr(cmd *exec.Cmd) {}

func terminateProcess(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}

func notifyContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(ctx, os.Interrupt)
}
