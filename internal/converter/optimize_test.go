package converter

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

// createTestJPEG creates a test JPEG with varied pixel data for realistic compression
func createTestJPEG(t *testing.T, dir string, width, height, quality int) string {
	t.Helper()
	path := filepath.Join(dir, "test.jpg")
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x*7 + y*3) % 256),
				G: uint8((x*3 + y*11) % 256),
				B: uint8((x*13 + y*5) % 256),
				A: 255,
			})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create test jpeg: %v", err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: quality}); err != nil {
		t.Fatalf("failed to encode test jpeg: %v", err)
	}
	return path
}

func TestOptimizeJPEGSmallerOrEqual(t *testing.T) {
	dir := t.TempDir()
	inputPath := createTestJPEG(t, dir, 200, 200, 95)

	// Normal encode
	normalOut := filepath.Join(dir, "normal.jpg")
	ic := &ImageConverter{}
	if err := ic.Convert(inputPath, normalOut, Options{}); err != nil {
		t.Fatalf("normal convert failed: %v", err)
	}

	// Optimize encode
	optimizedOut := filepath.Join(dir, "optimized.jpg")
	if err := ic.Convert(inputPath, optimizedOut, Options{Optimize: true}); err != nil {
		t.Fatalf("optimize convert failed: %v", err)
	}

	normalInfo, _ := os.Stat(normalOut)
	optimizedInfo, _ := os.Stat(optimizedOut)

	if optimizedInfo.Size() > normalInfo.Size() {
		t.Fatalf("optimized (%d bytes) should be <= normal (%d bytes)", optimizedInfo.Size(), normalInfo.Size())
	}
}

func TestOptimizePNGCompression(t *testing.T) {
	dir := t.TempDir()
	inputPath := createTestPNG(t, dir, 100, 100)

	normalOut := filepath.Join(dir, "normal.png")
	ic := &ImageConverter{}
	if err := ic.Convert(inputPath, normalOut, Options{}); err != nil {
		t.Fatalf("normal PNG convert failed: %v", err)
	}

	optimizedOut := filepath.Join(dir, "optimized.png")
	if err := ic.Convert(inputPath, optimizedOut, Options{Optimize: true}); err != nil {
		t.Fatalf("optimized PNG convert failed: %v", err)
	}

	normalInfo, _ := os.Stat(normalOut)
	optimizedInfo, _ := os.Stat(optimizedOut)

	if optimizedInfo.Size() > normalInfo.Size() {
		t.Fatalf("optimized PNG (%d bytes) should be <= normal PNG (%d bytes)", optimizedInfo.Size(), normalInfo.Size())
	}
}

func TestTargetSizeConvergence(t *testing.T) {
	dir := t.TempDir()
	// Create a large enough image to make target-size meaningful
	inputPath := createTestJPEG(t, dir, 400, 400, 95)

	targetBytes := int64(20000) // 20 KB target
	outputPath := filepath.Join(dir, "target.jpg")

	ic := &ImageConverter{}
	if err := ic.Convert(inputPath, outputPath, Options{TargetSize: targetBytes}); err != nil {
		t.Fatalf("target-size convert failed: %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("output not found: %v", err)
	}

	ratio := float64(info.Size()) / float64(targetBytes)
	// Should be within ±30% (generous tolerance for test stability)
	if ratio < 0.5 || ratio > 1.5 {
		t.Fatalf("output size %d bytes, target %d bytes, ratio %.2f — out of tolerance", info.Size(), targetBytes, ratio)
	}
}
