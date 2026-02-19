package cmd

import "testing"

func TestValidateTrimInput(t *testing.T) {
	if err := validateTrimInput("00:00:10", "5", "copy"); err == nil {
		t.Fatalf("expected error when end and duration both provided")
	}
	if err := validateTrimInput("", "", "invalid"); err == nil {
		t.Fatalf("expected error for invalid codec")
	}
	if err := validateTrimInput("", "5", "copy"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildTrimOutputPath(t *testing.T) {
	outputDir = ""
	got := buildTrimOutputPath("/tmp/in.mp4", "mp4", "", "")
	if got != "/tmp/in_trim.mp4" {
		t.Fatalf("unexpected output path: %s", got)
	}

	got = buildTrimOutputPath("/tmp/in.mp4", "mov", "cut", "")
	if got != "/tmp/cut.mov" {
		t.Fatalf("unexpected output path with custom name: %s", got)
	}
}
