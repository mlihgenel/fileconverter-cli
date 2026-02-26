package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestBuildMergeOutputPath(t *testing.T) {
	outputDir = ""
	got := buildMergeOutputPath("/tmp/part1.mp4", "mp4", "")
	if got != "/tmp/part1_merged.mp4" {
		t.Fatalf("unexpected output path: %s", got)
	}

	got = buildMergeOutputPath("/tmp/part1.mp4", "mov", "full_video")
	if got != "/tmp/full_video.mov" {
		t.Fatalf("unexpected output path with custom name: %s", got)
	}

	got = buildMergeOutputPath("/tmp/part1.mp4", "mkv", "")
	if got != "/tmp/part1_merged.mkv" {
		t.Fatalf("unexpected output path for mkv: %s", got)
	}
}

func TestMergeReencodeCodecArgs(t *testing.T) {
	// MP4 → libx264
	args := mergeReencodeCodecArgs("mp4", 80)
	foundH264 := false
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-c:v" && args[i+1] == "libx264" {
			foundH264 = true
		}
	}
	if !foundH264 {
		t.Fatalf("expected libx264 for mp4, got %v", args)
	}

	// WebM → libvpx-vp9
	args = mergeReencodeCodecArgs("webm", 80)
	foundVP9 := false
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-c:v" && args[i+1] == "libvpx-vp9" {
			foundVP9 = true
		}
	}
	if !foundVP9 {
		t.Fatalf("expected libvpx-vp9 for webm, got %v", args)
	}

	// AVI → mpeg4
	args = mergeReencodeCodecArgs("avi", 50)
	foundMpeg4 := false
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-c:v" && args[i+1] == "mpeg4" {
			foundMpeg4 = true
		}
	}
	if !foundMpeg4 {
		t.Fatalf("expected mpeg4 for avi, got %v", args)
	}
}

func TestWriteConcatList(t *testing.T) {
	tempDir := t.TempDir()
	files := []string{"/tmp/part1.mp4", "/tmp/part2.mp4"}
	listPath, err := writeConcatList(files, tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if listPath == "" {
		t.Fatalf("expected non-empty list path")
	}

	// Dosya içeriğini oku ve kontrol et
	data, err := readFileContent(listPath)
	if err != nil {
		t.Fatalf("could not read concat list: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "file '") {
		t.Fatalf("concat list should contain file entries, got: %s", content)
	}
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(lines))
	}
}

func readFileContent(path string) ([]byte, error) {
	return os.ReadFile(path)
}
