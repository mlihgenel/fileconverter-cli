package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mlihgenel/fileconverter-cli/internal/converter"
)

func TestVideoTrimIntegrationClipAndRemove(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped in short mode")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not found; skipping integration test")
	}

	tmpDir := t.TempDir()
	input := filepath.Join(tmpDir, "input.mp4")
	clipOut := filepath.Join(tmpDir, "clip.mp4")
	removeOut := filepath.Join(tmpDir, "remove.mp4")

	if err := generateIntegrationTestVideo(input, 6); err != nil {
		t.Fatalf("failed to generate test video: %v", err)
	}

	if err := runTrimFFmpeg(input, clipOut, "1", "", "2", "reencode", 70, converter.MetadataAuto, false); err != nil {
		t.Fatalf("runTrimFFmpeg failed: %v", err)
	}
	assertFileHasContent(t, clipOut)

	if err := runTrimRemoveFFmpeg(input, removeOut, "1", "", "2", "reencode", 70, converter.MetadataAuto, false); err != nil {
		t.Fatalf("runTrimRemoveFFmpeg failed: %v", err)
	}
	assertFileHasContent(t, removeOut)

	// ffprobe varsa yaklaşık süreyi de doğrulayalım.
	inDur, inOK := probeMediaDurationSeconds(input)
	clipDur, clipOK := probeMediaDurationSeconds(clipOut)
	removeDur, removeOK := probeMediaDurationSeconds(removeOut)
	if inOK && clipOK {
		if clipDur < 1.5 || clipDur > 2.5 {
			t.Fatalf("unexpected clip duration: %.3fs", clipDur)
		}
	}
	if inOK && removeOK {
		expected := inDur - 2.0
		if removeDur < expected-0.8 || removeDur > expected+0.8 {
			t.Fatalf("unexpected remove duration: got %.3fs expected around %.3fs", removeDur, expected)
		}
	}
}

func generateIntegrationTestVideo(output string, durationSec int) error {
	args := []string{
		"-loglevel", "error",
		"-f", "lavfi",
		"-i", "testsrc=size=320x240:rate=25",
		"-t", formatSecondsForFFmpeg(float64(durationSec)),
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-g", "1",
		"-an",
		"-y",
		output,
	}
	cmd := exec.Command("ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg test video generation failed: %s", string(out))
	}
	return nil
}

func assertFileHasContent(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("output not found: %s (%v)", path, err)
	}
	if info.Size() == 0 {
		t.Fatalf("output file is empty: %s", path)
	}
}
