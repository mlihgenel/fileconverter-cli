package converter

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// createTestPNG creates a simple test PNG in the given directory
func createTestPNG(t *testing.T, dir string, width, height int) string {
	t.Helper()
	path := filepath.Join(dir, "test.png")
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create test png: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("failed to encode test png: %v", err)
	}
	return path
}

func TestWebPEncodeFromPNG(t *testing.T) {
	dir := t.TempDir()
	inputPath := createTestPNG(t, dir, 64, 64)
	outputPath := filepath.Join(dir, "output.webp")

	ic := &ImageConverter{}
	err := ic.Convert(inputPath, outputPath, Options{})
	if err != nil {
		t.Fatalf("WebP encode failed: %v", err)
	}

	// Verify output file exists and has content
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}

	// Verify RIFF...WEBP header
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if len(data) < 12 {
		t.Fatal("output too small for RIFF header")
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		t.Fatalf("invalid WebP header: got %q...%q", string(data[0:4]), string(data[8:12]))
	}
}

func TestWebPRoundTrip(t *testing.T) {
	dir := t.TempDir()
	inputPath := createTestPNG(t, dir, 32, 32)

	// PNG -> WebP
	webpPath := filepath.Join(dir, "round.webp")
	ic := &ImageConverter{}
	if err := ic.Convert(inputPath, webpPath, Options{}); err != nil {
		t.Fatalf("PNG->WebP failed: %v", err)
	}

	// WebP -> PNG (round-trip)
	roundPath := filepath.Join(dir, "round.png")
	if err := ic.Convert(webpPath, roundPath, Options{}); err != nil {
		t.Fatalf("WebP->PNG failed: %v", err)
	}

	// Verify round-trip output
	info, err := os.Stat(roundPath)
	if err != nil {
		t.Fatalf("round-trip output not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("round-trip output is empty")
	}
}

func TestWebPToWebPNotInConversions(t *testing.T) {
	ic := &ImageConverter{}
	// webp -> webp should not appear in the supported conversions list
	for _, pair := range ic.SupportedConversions() {
		if pair.From == "webp" && pair.To == "webp" {
			t.Fatal("webp -> webp should not be listed in SupportedConversions")
		}
	}
}

func TestWebPInSupportedWriteFormats(t *testing.T) {
	ic := &ImageConverter{}
	// Verify webp is listed as a target format
	found := false
	for _, pair := range ic.SupportedConversions() {
		if pair.To == "webp" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("webp should be in supported target formats")
	}
}
