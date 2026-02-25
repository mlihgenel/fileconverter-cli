package converter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetFileInfoImage(t *testing.T) {
	dir := t.TempDir()
	path := createTestPNG(t, dir, 120, 80)

	info, err := GetFileInfo(path)
	if err != nil {
		t.Fatalf("GetFileInfo failed: %v", err)
	}

	if info.Format != "PNG" {
		t.Fatalf("expected format PNG, got %s", info.Format)
	}
	if info.Category != "image" {
		t.Fatalf("expected category image, got %s", info.Category)
	}
	if info.Width != 120 || info.Height != 80 {
		t.Fatalf("expected 120x80, got %dx%d", info.Width, info.Height)
	}
	if info.Resolution != "120x80" {
		t.Fatalf("expected resolution 120x80, got %s", info.Resolution)
	}
	if info.Size <= 0 {
		t.Fatal("expected positive file size")
	}
	if info.SizeText == "" {
		t.Fatal("expected non-empty size text")
	}
}

func TestGetFileInfoMissingFile(t *testing.T) {
	_, err := GetFileInfo("/nonexistent/file.png")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestCategorizeFormat(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"png", "image"},
		{"jpg", "image"},
		{"webp", "image"},
		{"mp4", "video"},
		{"mov", "video"},
		{"mp3", "audio"},
		{"flac", "audio"},
		{"pdf", "document"},
		{"md", "document"},
		{"xyz", "unknown"},
	}

	for _, tt := range tests {
		got := categorizeFormat(tt.format)
		if got != tt.expected {
			t.Errorf("categorizeFormat(%q) = %q, want %q", tt.format, got, tt.expected)
		}
	}
}

func TestGetFileInfoJPEG(t *testing.T) {
	dir := t.TempDir()
	path := createTestJPEG(t, dir, 200, 150, 85)

	// Rename to ensure correct extension detection
	jpgPath := filepath.Join(dir, "photo.jpg")
	if err := copyFile(path, jpgPath); err != nil {
		t.Fatalf("failed to copy: %v", err)
	}

	info, err := GetFileInfo(jpgPath)
	if err != nil {
		t.Fatalf("GetFileInfo failed: %v", err)
	}

	if info.Category != "image" {
		t.Fatalf("expected category image, got %s", info.Category)
	}
	if info.Width != 200 || info.Height != 150 {
		t.Fatalf("expected 200x150, got %dx%d", info.Width, info.Height)
	}
}

// copyFile is a simple file copy helper for tests
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
