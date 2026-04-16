package watch

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	watcher   *fsnotify.Watcher
	vaultPath string
}

func New(vaultPath string) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("watch: new watcher: %w", err)
	}

	return &Watcher{
		watcher:   w,
		vaultPath: vaultPath,
	}, nil
}

func (w *Watcher) Add(path string) error {
	return w.watcher.Add(path)
}

func (w *Watcher) Events() <-chan fsnotify.Event {
	return w.watcher.Events
}

func (w *Watcher) Errors() <-chan error {
	return w.watcher.Errors
}

func (w *Watcher) Close() error {
	return w.watcher.Close()
}
