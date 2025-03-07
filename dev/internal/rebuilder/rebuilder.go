package rebuilder

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func Serve(ctx context.Context) error {
	entries, err := readProcfile("Procfile")
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	reloadCh := make([]chan bool, len(entries))
	exitCh := make(chan error, len(entries))

	go new(watcher).Watch(reloadCh)
	for i, e := range entries {
		reloadCh[i] = make(chan bool)
		go func() {
			exitCh <- newProcess(e).Run(ctx, reloadCh[i])
		}()
	}

	<-ctx.Done()

	for range entries {
		<-exitCh
	}

	return nil
}
