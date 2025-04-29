package rebuilder

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
)

var watchExtensions string

func init() {
	pflag.StringVar(&watchExtensions, "watch.extensions", ".go", "Comma-separated list of file extensions to watch for changes and trigger recompilation (e.g. .go,.css,.js).")
}

type watcher struct {
	watcher *fsnotify.Watcher
}

func (w *watcher) Watch(reloadCh []chan bool) {
	pflag.Parse()

	var err error

	w.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[error] error creating watcher: %v\n", err)
		return
	}

	defer w.watcher.Close()

	w.add(".")

	d := newDebounce()
	defer d.timer.Stop()

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			if event.Has(fsnotify.Create) {
				w.add(event.Name)
			}

			if event.Has(fsnotify.Remove) {
				w.remove(event.Name)
			}

			if !slices.Contains(strings.Split(watchExtensions, ","), filepath.Ext(event.Name)) {
				continue
			}

			d.Trigger(reloadCh)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
	}
}

func (w *watcher) add(path string) {
	filepath.WalkDir(path, func(dir string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() {
			return nil
		}

		w.watcher.Add(dir)

		return nil
	})
}

func (w *watcher) remove(path string) {
	filepath.WalkDir(path, func(dir string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		w.watcher.Remove(dir)

		return nil
	})
}

func newDebounce() *debounce {
	return &debounce{
		delay: 100 * time.Millisecond,
	}
}

type debounce struct {
	timer *time.Timer
	delay time.Duration
}

func (d *debounce) Trigger(reloadCh []chan bool) {
	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.delay, func() {
		for _, ch := range reloadCh {
			ch <- true
		}
	})
}
