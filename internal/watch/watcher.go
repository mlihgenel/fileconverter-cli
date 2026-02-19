package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mlihgenel/fileconverter-cli/internal/converter"
)

type fileState struct {
	Size       int64
	ModTime    time.Time
	LastChange time.Time
	Processed  bool
}

// Watcher polling tabanlı dosya izleyicisidir.
type Watcher struct {
	Root      string
	From      string
	Recursive bool
	SettleFor time.Duration

	states map[string]fileState
}

// NewWatcher yeni bir watcher oluşturur.
func NewWatcher(root, from string, recursive bool, settleFor time.Duration) *Watcher {
	if settleFor <= 0 {
		settleFor = 1500 * time.Millisecond
	}
	return &Watcher{
		Root:      root,
		From:      converter.NormalizeFormat(from),
		Recursive: recursive,
		SettleFor: settleFor,
		states:    make(map[string]fileState),
	}
}

// Bootstrap mevcut dosyaları "zaten işlenmiş" olarak kaydeder.
func (w *Watcher) Bootstrap() error {
	now := time.Now()
	return w.scan(func(path string, info os.FileInfo) error {
		w.states[path] = fileState{
			Size:       info.Size(),
			ModTime:    info.ModTime(),
			LastChange: now,
			Processed:  true,
		}
		return nil
	})
}

// Poll yeni/degisen ve stabilize olmuş dosyaları döner.
func (w *Watcher) Poll(now time.Time) ([]string, error) {
	seen := make(map[string]struct{})
	var ready []string

	err := w.scan(func(path string, info os.FileInfo) error {
		seen[path] = struct{}{}
		state, ok := w.states[path]

		if !ok {
			w.states[path] = fileState{
				Size:       info.Size(),
				ModTime:    info.ModTime(),
				LastChange: now,
				Processed:  false,
			}
			return nil
		}

		if state.Size != info.Size() || !state.ModTime.Equal(info.ModTime()) {
			state.Size = info.Size()
			state.ModTime = info.ModTime()
			state.LastChange = now
			state.Processed = false
			w.states[path] = state
			return nil
		}

		if !state.Processed && now.Sub(state.LastChange) >= w.SettleFor {
			state.Processed = true
			w.states[path] = state
			ready = append(ready, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for path := range w.states {
		if _, ok := seen[path]; !ok {
			delete(w.states, path)
		}
	}

	return ready, nil
}

func (w *Watcher) scan(onFile func(path string, info os.FileInfo) error) error {
	info, err := os.Stat(w.Root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("watch yolu dizin olmalidir: %s", w.Root)
	}

	walkFn := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if !w.Recursive && path != w.Root {
				return filepath.SkipDir
			}
			return nil
		}
		if !converter.HasFormatExtension(path, w.From) {
			return nil
		}
		info, statErr := d.Info()
		if statErr != nil {
			return nil
		}
		return onFile(path, info)
	}

	return filepath.WalkDir(w.Root, walkFn)
}
