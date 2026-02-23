package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// EventWatcher fsnotify ile event-driven izleme sağlar.
type EventWatcher struct {
	poller *Watcher
	fs     *fsnotify.Watcher

	root      string
	recursive bool

	events chan struct{}
	done   chan struct{}
	once   sync.Once
}

// NewEventWatcher fsnotify backend'i oluşturur.
func NewEventWatcher(root, from string, recursive bool, settleFor time.Duration) (*EventWatcher, error) {
	fs, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	ew := &EventWatcher{
		poller:    NewWatcher(root, from, recursive, settleFor),
		fs:        fs,
		root:      root,
		recursive: recursive,
		events:    make(chan struct{}, 1),
		done:      make(chan struct{}),
	}
	return ew, nil
}

// NewAdaptiveWatcher event backend'i dener; olmazsa polling fallback döner.
func NewAdaptiveWatcher(root, from string, recursive bool, settleFor time.Duration) (Engine, error) {
	eventWatcher, err := NewEventWatcher(root, from, recursive, settleFor)
	if err != nil {
		return NewWatcher(root, from, recursive, settleFor), err
	}
	return eventWatcher, nil
}

func (w *EventWatcher) Bootstrap() error {
	if err := w.poller.Bootstrap(); err != nil {
		return err
	}
	if err := w.watchDirectories(); err != nil {
		return err
	}

	go w.loop()
	return nil
}

func (w *EventWatcher) Poll(now time.Time) ([]string, error) {
	return w.poller.Poll(now)
}

func (w *EventWatcher) Events() <-chan struct{} {
	return w.events
}

func (w *EventWatcher) Close() error {
	w.once.Do(func() {
		close(w.done)
	})
	return w.fs.Close()
}

func (w *EventWatcher) Mode() string { return "event+polling" }

func (w *EventWatcher) loop() {
	for {
		select {
		case <-w.done:
			return
		case evt, ok := <-w.fs.Events:
			if !ok {
				return
			}
			if evt.Has(fsnotify.Create) && w.recursive {
				if info, err := os.Stat(evt.Name); err == nil && info.IsDir() {
					_ = w.addWatchPath(evt.Name)
				}
			}
			w.signal()
		case <-w.fs.Errors:
			// Event tabanlı backend hata alsa da polling devam ettiği için sessiz geç.
			w.signal()
		}
	}
}

func (w *EventWatcher) signal() {
	select {
	case w.events <- struct{}{}:
	default:
	}
}

func (w *EventWatcher) watchDirectories() error {
	info, err := os.Stat(w.root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("watch yolu dizin olmalidir: %s", w.root)
	}

	if !w.recursive {
		return w.addWatchPath(w.root)
	}

	walkErr := filepath.WalkDir(w.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		return w.addWatchPath(path)
	})
	if walkErr != nil {
		return walkErr
	}
	return nil
}

func (w *EventWatcher) addWatchPath(path string) error {
	return w.fs.Add(path)
}
