package cmd

import "testing"

func TestBuildSnapshotOutputPath(t *testing.T) {
	outputDir = ""
	got := buildSnapshotOutputPath("/tmp/video.mp4", "png", "", 30)
	if got != "/tmp/video_snapshot_30s.png" {
		t.Fatalf("unexpected output path: %s", got)
	}

	got = buildSnapshotOutputPath("/tmp/video.mp4", "jpg", "thumbnail", 60)
	if got != "/tmp/thumbnail.jpg" {
		t.Fatalf("unexpected output path with custom name: %s", got)
	}

	got = buildSnapshotOutputPath("/tmp/video.mp4", "webp", "", 90.5)
	if got != "/tmp/video_snapshot_90s.webp" {
		t.Fatalf("unexpected output path: %s", got)
	}
}

func TestIsValidSnapshotFormat(t *testing.T) {
	valid := []string{"png", "jpg", "webp", "bmp"}
	for _, f := range valid {
		if !isValidSnapshotFormat(f) {
			t.Fatalf("expected %s to be valid", f)
		}
	}
	invalid := []string{"mp4", "mp3", "pdf", "gif", ""}
	for _, f := range invalid {
		if isValidSnapshotFormat(f) {
			t.Fatalf("expected %s to be invalid", f)
		}
	}
}

func TestSnapshotCodecArgs(t *testing.T) {
	// PNG — boş args
	args := snapshotCodecArgs("png", 80)
	if len(args) != 0 {
		t.Fatalf("expected empty args for png, got %v", args)
	}

	// JPG quality
	args = snapshotCodecArgs("jpg", 90)
	if len(args) != 2 || args[0] != "-q:v" {
		t.Fatalf("expected qscale args for jpg, got %v", args)
	}

	// WebP quality
	args = snapshotCodecArgs("webp", 85)
	if len(args) != 2 || args[0] != "-quality" || args[1] != "85" {
		t.Fatalf("expected quality args for webp, got %v", args)
	}
}

func TestResolveSnapshotTimeSeconds(t *testing.T) {
	// Saniye
	got, err := resolveSnapshotTime("30", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 30 {
		t.Fatalf("expected 30, got %f", got)
	}

	// HH:MM:SS
	got, err = resolveSnapshotTime("00:01:30", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 90 {
		t.Fatalf("expected 90, got %f", got)
	}
}

func TestResolveSnapshotTimeInvalid(t *testing.T) {
	// Boş
	if _, err := resolveSnapshotTime("", ""); err == nil {
		t.Fatalf("expected error for empty time")
	}

	// Geçersiz yüzde
	if _, err := resolveSnapshotTime("%abc", ""); err == nil {
		t.Fatalf("expected error for invalid percentage")
	}

	// Yüzde > 100
	if _, err := resolveSnapshotTime("%150", ""); err == nil {
		t.Fatalf("expected error for percentage > 100")
	}
}
