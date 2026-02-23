package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mlihgenel/fileconverter-cli/internal/converter"
)

func TestResolveBatchOutputPathSkip(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "out.jpg")
	if err := os.WriteFile(base, []byte("x"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	resolved, reason, err := resolveBatchOutputPath(base, converter.ConflictSkip, map[string]struct{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != base {
		t.Fatalf("unexpected resolved path: %s", resolved)
	}
	if reason != "output_exists" {
		t.Fatalf("expected output_exists, got %s", reason)
	}
}

func TestResolveBatchOutputPathVersioned(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "out.jpg")
	if err := os.WriteFile(base, []byte("x"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "out (1).jpg"), []byte("x"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	reserved := map[string]struct{}{
		filepath.Join(dir, "out (2).jpg"): {},
	}
	resolved, reason, err := resolveBatchOutputPath(base, converter.ConflictVersioned, reserved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reason != "" {
		t.Fatalf("unexpected skip reason: %s", reason)
	}
	want := filepath.Join(dir, "out (3).jpg")
	if resolved != want {
		t.Fatalf("expected %s, got %s", want, resolved)
	}
}

func TestResolveBatchOutputPathOverwrite(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "out.jpg")

	reserved := map[string]struct{}{}
	resolved, reason, err := resolveBatchOutputPath(base, converter.ConflictOverwrite, reserved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reason != "" {
		t.Fatalf("unexpected skip reason: %s", reason)
	}
	if resolved != base {
		t.Fatalf("unexpected resolved path: %s", resolved)
	}
}

func TestBuildBatchOutputPathPreserveTree(t *testing.T) {
	prevOutput := outputDir
	prevPreserve := batchPreserveTree
	t.Cleanup(func() {
		outputDir = prevOutput
		batchPreserveTree = prevPreserve
	})

	outputDir = filepath.Join("target", "out")
	batchPreserveTree = true

	input := filepath.Join("src", "nested", "asset.jpg")
	got := buildBatchOutputPath(input, filepath.Join("src"), "png")
	want := filepath.Join("target", "out", "nested", "asset.png")
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestBuildBatchOutputPathPreserveTreeFallback(t *testing.T) {
	prevOutput := outputDir
	prevPreserve := batchPreserveTree
	t.Cleanup(func() {
		outputDir = prevOutput
		batchPreserveTree = prevPreserve
	})

	outputDir = filepath.Join("target", "out")
	batchPreserveTree = true

	input := filepath.Join("other", "asset.jpg")
	got := buildBatchOutputPath(input, filepath.Join("src"), "png")
	want := filepath.Join("target", "out", "asset.png")
	if got != want {
		t.Fatalf("expected fallback path %s, got %s", want, got)
	}
}
