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

func TestBuildVideoTrimExecutionClip(t *testing.T) {
	m := newInteractiveModel(nil, false)
	m.selectedFile = "/tmp/sample.mp4"
	m.trimMode = trimModeClip
	m.trimRangeType = trimRangeDuration
	m.trimStartInput = "5"
	m.trimDurationInput = "2"
	m.trimCodec = "copy"
	m.defaultOutput = "/tmp"

	execution, err := m.buildVideoTrimExecution()
	if err != nil {
		t.Fatalf("unexpected execution build error: %v", err)
	}
	if execution.TargetFormat != "mp4" {
		t.Fatalf("expected target format mp4, got %s", execution.TargetFormat)
	}
	if execution.Plan.Mode != trimModeClip {
		t.Fatalf("expected clip plan mode, got %s", execution.Plan.Mode)
	}
	if !execution.Plan.ClipHasEnd || execution.Plan.ClipStartSec != 5 || execution.Plan.ClipEndSec != 7 {
		t.Fatalf("unexpected clip plan values: %+v", execution.Plan)
	}
}

func TestBuildVideoTrimExecutionRemove(t *testing.T) {
	m := newInteractiveModel(nil, false)
	m.selectedFile = "/tmp/sample.mp4"
	m.trimMode = trimModeRemove
	m.trimRangeType = trimRangeEnd
	m.trimStartInput = "23"
	m.trimEndInput = "25"
	m.trimCodec = "copy"
	m.defaultOutput = "/tmp"

	execution, err := m.buildVideoTrimExecution()
	if err != nil {
		t.Fatalf("unexpected execution build error: %v", err)
	}
	if execution.Plan.Mode != trimModeRemove {
		t.Fatalf("expected remove plan mode, got %s", execution.Plan.Mode)
	}
	if len(execution.Plan.RemoveRanges) != 1 {
		t.Fatalf("expected 1 remove range, got %d", len(execution.Plan.RemoveRanges))
	}
	if execution.Plan.RemoveRanges[0].Start != 23 || execution.Plan.RemoveRanges[0].End != 25 {
		t.Fatalf("unexpected remove range: %+v", execution.Plan.RemoveRanges[0])
	}
}

func TestPrepareVideoTrimTimelineAndAdjust(t *testing.T) {
	m := newInteractiveModel(nil, false)
	m.selectedFile = "/tmp/sample.mp4"
	m.trimMode = trimModeClip
	m.trimRangeType = trimRangeDuration
	m.trimStartInput = "10"
	m.trimDurationInput = "5"

	if err := m.prepareVideoTrimTimeline(); err != nil {
		t.Fatalf("unexpected timeline prepare error: %v", err)
	}
	if m.trimTimelineStart != 10 || m.trimTimelineEnd != 15 {
		t.Fatalf("unexpected timeline bounds: start=%.2f end=%.2f", m.trimTimelineStart, m.trimTimelineEnd)
	}

	m.cursor = 0
	m.trimTimelineStep = 1
	m.adjustVideoTrimTimeline(2)
	if m.trimTimelineStart != 12 {
		t.Fatalf("expected start to move to 12, got %.2f", m.trimTimelineStart)
	}

	m.cursor = 1
	m.adjustVideoTrimTimeline(-1)
	if m.trimTimelineEnd != 14 {
		t.Fatalf("expected end to move to 14, got %.2f", m.trimTimelineEnd)
	}
}

func TestTimelineStepHelpers(t *testing.T) {
	if got := increaseTimelineStep(1); got != 2 {
		t.Fatalf("expected increase from 1 to 2, got %.1f", got)
	}
	if got := decreaseTimelineStep(1); got != 0.5 {
		t.Fatalf("expected decrease from 1 to 0.5, got %.1f", got)
	}
}
