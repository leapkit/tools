package rebuilder

import (
	"context"
)

func Serve(ctx context.Context) error {
	entries, err := readProcfile("Procfile")
	if err != nil {
		return err
	}

	ctx, cancel := notifyContext(ctx)
	defer cancel()

	reloadCh := make([]chan bool, len(entries))
	for i := range reloadCh {
		reloadCh[i] = make(chan bool)
	}

	errCh := make(chan error, len(entries))

	go new(watcher).Watch(ctx, reloadCh)
	for i, e := range entries {
		go func() {
			errCh <- newProcess(e).Run(ctx, reloadCh[i])
		}()
	}

	<-ctx.Done()

	var wErr error
	for range entries {
		if err := <-errCh; err != nil && wErr == nil {
			wErr = err
		}
	}

	return wErr
}
