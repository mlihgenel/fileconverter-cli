package converter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeConflictPolicy(t *testing.T) {
	if got := NormalizeConflictPolicy(""); got != ConflictVersioned {
		t.Fatalf("expected default policy %s, got %s", ConflictVersioned, got)
	}
	if got := NormalizeConflictPolicy("OVERWRITE"); got != ConflictOverwrite {
		t.Fatalf("expected overwrite, got %s", got)
	}
	if got := NormalizeConflictPolicy("bad"); got != "" {
		t.Fatalf("expected empty for invalid policy, got %s", got)
	}
}

func TestResolveOutputPathConflictOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.jpg")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	got, skip, err := ResolveOutputPathConflict(path, ConflictOverwrite)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skip {
		t.Fatalf("overwrite should not skip")
	}
	if got != path {
		t.Fatalf("unexpected resolved path: %s", got)
	}
}

func TestResolveOutputPathConflictSkip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.jpg")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	got, skip, err := ResolveOutputPathConflict(path, ConflictSkip)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !skip {
		t.Fatalf("skip policy should skip")
	}
	if got != path {
		t.Fatalf("unexpected resolved path: %s", got)
	}
}

func TestResolveOutputPathConflictVersioned(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.jpg")
	path1 := filepath.Join(dir, "out (1).jpg")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := os.WriteFile(path1, []byte("x"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	got, skip, err := ResolveOutputPathConflict(path, ConflictVersioned)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skip {
		t.Fatalf("versioned should not skip")
	}
	want := filepath.Join(dir, "out (2).jpg")
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestResolveOutputPathConflictInvalidPolicy(t *testing.T) {
	_, _, err := ResolveOutputPathConflict("test.jpg", "invalid")
	if err == nil {
		t.Fatalf("expected error for invalid policy")
	}
}
