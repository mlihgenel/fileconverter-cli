package cmd

import "testing"

func TestValidateTrimInput(t *testing.T) {
	if err := validateTrimInput("clip", "00:00:10", "5", "", "copy"); err == nil {
		t.Fatalf("expected error when end and duration both provided")
	}
	if err := validateTrimInput("clip", "", "", "", "invalid"); err == nil {
		t.Fatalf("expected error for invalid codec")
	}
	if err := validateTrimInput("clip", "", "5", "", "copy"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateTrimInput("remove", "", "", "", "copy"); err == nil {
		t.Fatalf("expected error when remove mode has no end/duration")
	}
	if err := validateTrimInput("clip", "", "", "1-2", "copy"); err == nil {
		t.Fatalf("expected error when ranges is used outside remove mode")
	}
	if err := validateTrimInput("remove", "", "", "1-2", "copy"); err != nil {
		t.Fatalf("unexpected error for remove+valid ranges: %v", err)
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

func TestParseTrimRangesSpecMergesRanges(t *testing.T) {
	ranges, err := parseTrimRangesSpec("00:00:05-00:00:08,00:00:07-00:00:10,20-22")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(ranges) != 2 {
		t.Fatalf("expected merged ranges length 2, got %d", len(ranges))
	}
	if ranges[0].Start != 5 || ranges[0].End != 10 {
		t.Fatalf("unexpected first range: %+v", ranges[0])
	}
	if ranges[1].Start != 20 || ranges[1].End != 22 {
		t.Fatalf("unexpected second range: %+v", ranges[1])
	}
}

func TestParseTrimRangesSpecRejectsInvalid(t *testing.T) {
	if _, err := parseTrimRangesSpec("bad"); err == nil {
		t.Fatalf("expected invalid range format error")
	}
	if _, err := parseTrimRangesSpec("10-5"); err == nil {
		t.Fatalf("expected invalid reversed range error")
	}
}

func TestBuildKeepSegmentsFromRanges(t *testing.T) {
	segments, err := buildKeepSegmentsFromRanges([]trimRange{
		{Start: 5, End: 10},
		{Start: 20, End: 22},
	}, 30, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(segments) != 3 {
		t.Fatalf("expected 3 keep segments, got %d", len(segments))
	}
	if segments[0].Start != 0 || segments[0].End != 5 {
		t.Fatalf("unexpected first segment: %+v", segments[0])
	}
	if segments[2].Start != 22 || segments[2].End != 30 {
		t.Fatalf("unexpected last segment: %+v", segments[2])
	}
}

func TestResolveRemoveRanges(t *testing.T) {
	ranges, err := resolveRemoveRanges("00:00:23", "", "2", nil)
	if err != nil {
		t.Fatalf("unexpected resolve error: %v", err)
	}
	if len(ranges) != 1 {
		t.Fatalf("expected single range, got %d", len(ranges))
	}
	if ranges[0].Start != 23 || ranges[0].End != 25 {
		t.Fatalf("unexpected range: %+v", ranges[0])
	}

	ranges, err = resolveRemoveRanges("", "", "", []trimRange{
		{Start: 10, End: 12},
		{Start: 11.5, End: 13},
	})
	if err != nil {
		t.Fatalf("unexpected explicit range error: %v", err)
	}
	if len(ranges) != 1 || ranges[0].Start != 10 || ranges[0].End != 13 {
		t.Fatalf("expected merged explicit range, got %+v", ranges)
	}
}

func TestResolveRemoveRangesRejectsInvalid(t *testing.T) {
	if _, err := resolveRemoveRanges("10", "5", "", nil); err == nil {
		t.Fatalf("expected error when end <= start")
	}
	if _, err := resolveRemoveRanges("bad", "", "2", nil); err == nil {
		t.Fatalf("expected parse error for invalid start")
	}
}

func TestSumKeepSegmentsLength(t *testing.T) {
	total, known := sumKeepSegmentsLength([]keepSegment{
		{Start: 0, End: 5, HasEnd: true},
		{Start: 10, End: 12, HasEnd: true},
	})
	if !known {
		t.Fatalf("expected known=true")
	}
	if total != 7 {
		t.Fatalf("expected total=7, got %.2f", total)
	}

	_, known = sumKeepSegmentsLength([]keepSegment{
		{Start: 12, HasEnd: false},
	})
	if known {
		t.Fatalf("expected known=false for open-ended segment")
	}
}

func TestFormatTrimSecondsHuman(t *testing.T) {
	if got := formatTrimSecondsHuman(65); got != "00:01:05" {
		t.Fatalf("unexpected formatted value: %s", got)
	}
	if got := formatTrimSecondsHuman(65.345); got != "00:01:05.345" {
		t.Fatalf("unexpected formatted millis value: %s", got)
	}
}
