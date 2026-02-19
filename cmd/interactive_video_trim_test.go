package cmd

import "testing"

func TestNormalizeVideoTrimTime(t *testing.T) {
	got, err := normalizeVideoTrimTime("5,5", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "5.5" {
		t.Fatalf("unexpected normalized value: %s", got)
	}

	got, err = normalizeVideoTrimTime("00:01:05", false)
	if err != nil {
		t.Fatalf("unexpected error for hh:mm:ss: %v", err)
	}
	if got != "00:01:05" {
		t.Fatalf("unexpected hh:mm:ss normalization: %s", got)
	}

	if _, err := normalizeVideoTrimTime("0", false); err == nil {
		t.Fatalf("expected error when duration is zero")
	}
	if _, err := normalizeVideoTrimTime("-1", true); err == nil {
		t.Fatalf("expected error for negative value")
	}
}

func TestParseVideoTrimToSeconds(t *testing.T) {
	sec, err := parseVideoTrimToSeconds("01:02:03")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sec != 3723 {
		t.Fatalf("unexpected seconds value: %.2f", sec)
	}

	sec, err = parseVideoTrimToSeconds("10:30")
	if err != nil {
		t.Fatalf("unexpected error for mm:ss: %v", err)
	}
	if sec != 630 {
		t.Fatalf("unexpected mm:ss conversion: %.2f", sec)
	}

	if _, err := parseVideoTrimToSeconds("00:70"); err == nil {
		t.Fatalf("expected error for invalid seconds part")
	}
}

func TestIsVideoTrimSourceFile(t *testing.T) {
	if !isVideoTrimSourceFile("clip.mp4") {
		t.Fatalf("expected mp4 to be accepted")
	}
	if isVideoTrimSourceFile("notes.txt") {
		t.Fatalf("expected txt to be rejected")
	}
}
