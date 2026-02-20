package cmd

import "testing"

func TestValidateTrimInput(t *testing.T) {
	if err := validateTrimInput("clip", "00:00:10", "5", "copy"); err == nil {
		t.Fatalf("expected error when end and duration both provided")
	}
	if err := validateTrimInput("clip", "", "", "invalid"); err == nil {
		t.Fatalf("expected error for invalid codec")
	}
	if err := validateTrimInput("clip", "", "5", "copy"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateTrimInput("remove", "", "", "copy"); err == nil {
		t.Fatalf("expected error when remove mode has no end/duration")
	}
}

func TestBuildTrimOutputPath(t *testing.T) {
	outputDir = ""
	got := buildTrimOutputPath("/tmp/in.mp4", "mp4", "", "", trimModeClip)
	if got != "/tmp/in_trim.mp4" {
		t.Fatalf("unexpected output path: %s", got)
	}

	got = buildTrimOutputPath("/tmp/in.mp4", "mov", "cut", "", trimModeClip)
	if got != "/tmp/cut.mov" {
		t.Fatalf("unexpected output path with custom name: %s", got)
	}

	got = buildTrimOutputPath("/tmp/in.mp4", "mp4", "", "", trimModeRemove)
	if got != "/tmp/in_cut.mp4" {
		t.Fatalf("unexpected output path for remove mode: %s", got)
	}
}

func TestResolveTrimRange(t *testing.T) {
	start, end, duration, _, _, err := resolveTrimRange("00:00:23", "", "2", trimModeRemove)
	if err != nil {
		t.Fatalf("unexpected resolve error: %v", err)
	}
	if start != "00:00:23" || duration != "2" || end != "" {
		t.Fatalf("unexpected values: start=%s end=%s duration=%s", start, end, duration)
	}

	if _, _, _, _, _, err := resolveTrimRange("10", "5", "", trimModeClip); err == nil {
		t.Fatalf("expected range error when end <= start")
	}
}

func TestClampTrimWindowToDuration(t *testing.T) {
	start, end, err := clampTrimWindowToDuration(23, 25, 60, trimModeRemove)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start != 23 || end != 25 {
		t.Fatalf("unexpected unchanged values: start=%.2f end=%.2f", start, end)
	}

	_, end, err = clampTrimWindowToDuration(55, 70, 60, trimModeRemove)
	if err != nil {
		t.Fatalf("unexpected error on clamp: %v", err)
	}
	if end != 60 {
		t.Fatalf("expected end to clamp to duration, got %.2f", end)
	}

	if _, _, err := clampTrimWindowToDuration(60, 61, 60, trimModeClip); err == nil {
		t.Fatalf("expected error when start is out of duration")
	}

	if _, _, err := clampTrimWindowToDuration(10, 10, 60, trimModeRemove); err == nil {
		t.Fatalf("expected error when end <= start")
	}
}
