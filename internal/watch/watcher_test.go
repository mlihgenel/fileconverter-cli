package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherBootstrapAndPoll(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "existing.jpg")
	if err := os.WriteFile(existing, []byte("old"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	w := NewWatcher(dir, "jpg", false, time.Second)
	if err := w.Bootstrap(); err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	now := time.Now()
	ready, err := w.Poll(now)
	if err != nil {
		t.Fatalf("poll failed: %v", err)
	}
	if len(ready) != 0 {
		t.Fatalf("expected no ready files after bootstrap")
	}

	newFile := filepath.Join(dir, "new.jpg")
	if err := os.WriteFile(newFile, []byte("new"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	ready, err = w.Poll(now.Add(100 * time.Millisecond))
	if err != nil {
		t.Fatalf("poll failed: %v", err)
	}
	if len(ready) != 0 {
		t.Fatalf("expected no ready files before settle")
	}

	ready, err = w.Poll(now.Add(2 * time.Second))
	if err != nil {
		t.Fatalf("poll failed: %v", err)
	}
	if len(ready) != 1 || ready[0] != newFile {
		t.Fatalf("expected new file ready, got: %#v", ready)
	}

	ready, err = w.Poll(now.Add(3 * time.Second))
	if err != nil {
		t.Fatalf("poll failed: %v", err)
	}
	if len(ready) != 0 {
		t.Fatalf("expected file to be emitted once")
	}
}

func TestWatcherDetectsModifiedFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "mod.jpg")
	if err := os.WriteFile(f, []byte("a"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	w := NewWatcher(dir, "jpg", false, 500*time.Millisecond)
	if err := w.Bootstrap(); err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	base := time.Now()
	if err := os.WriteFile(f, []byte("changed-content"), 0644); err != nil {
		t.Fatalf("rewrite failed: %v", err)
	}

	ready, err := w.Poll(base.Add(100 * time.Millisecond))
	if err != nil {
		t.Fatalf("poll failed: %v", err)
	}
	if len(ready) != 0 {
		t.Fatalf("expected no ready file before settle")
	}

	ready, err = w.Poll(base.Add(2 * time.Second))
	if err != nil {
		t.Fatalf("poll failed: %v", err)
	}
	if len(ready) != 1 || ready[0] != f {
		t.Fatalf("expected modified file ready once, got: %#v", ready)
	}
}
