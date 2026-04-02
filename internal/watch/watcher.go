package watch

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors session files for changes
type Watcher struct {
	watcher  *fsnotify.Watcher
	sessions map[string]bool
	onChange func(path string)
}

// NewWatcher creates a new file watcher
func NewWatcher(onChange func(path string)) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}

	return &Watcher{
		watcher:  fw,
		sessions: make(map[string]bool),
		onChange: onChange,
	}, nil
}

// WatchSessionDir adds a session directory to watch
func (w *Watcher) WatchSessionDir(dir string) error {
	if err := w.watcher.Add(dir); err != nil {
		return fmt.Errorf("watch %s: %w", dir, err)
	}
	return nil
}

// Start begins watching for changes
func (w *Watcher) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					if filepath.Ext(event.Name) == ".jsonl" {
						w.onChange(event.Name)
					}
				}
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Watcher error: %v", err)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop stops the watcher
func (w *Watcher) Stop() error {
	return w.watcher.Close()
}
