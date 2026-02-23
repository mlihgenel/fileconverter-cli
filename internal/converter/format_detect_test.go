package converter

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectFormatFallbackToExtension(t *testing.T) {
	got := DetectFormat("missing-file.mp4")
	if got != "mp4" {
		t.Fatalf("expected mp4, got %s", got)
	}
}

func TestDetectFormatUsesContentForWrongExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "image.txt")
	if err := os.WriteFile(path, []byte{
		0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n',
		0, 0, 0, 0,
	}, 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	got := DetectFormat(path)
	if got != "png" {
		t.Fatalf("expected png, got %s", got)
	}
}

func TestDetectFormatPrefersTextExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "table.csv")
	if err := os.WriteFile(path, []byte("id,name\n1,ali\n"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	got := DetectFormat(path)
	if got != "csv" {
		t.Fatalf("expected csv, got %s", got)
	}
}

func TestDetectFormatDetectsDocxByZipContents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.bin")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("zip create failed: %v", err)
	}
	if _, err := w.Write([]byte("<w:document/>")); err != nil {
		t.Fatalf("zip write failed: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close failed: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	got := DetectFormat(path)
	if got != "docx" {
		t.Fatalf("expected docx, got %s", got)
	}
}
