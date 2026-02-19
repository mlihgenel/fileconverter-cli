package batch

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNormalizeReportFormat(t *testing.T) {
	if got := NormalizeReportFormat(""); got != ReportOff {
		t.Fatalf("expected off, got %s", got)
	}
	if got := NormalizeReportFormat("JSON"); got != ReportJSON {
		t.Fatalf("expected json, got %s", got)
	}
	if got := NormalizeReportFormat("bad"); got != "" {
		t.Fatalf("expected empty for invalid report format, got %s", got)
	}
}

func TestRenderReportTXT(t *testing.T) {
	summary := Summary{
		Total:     2,
		Succeeded: 1,
		Skipped:   1,
		Failed:    0,
		Duration:  2 * time.Second,
	}
	results := []JobResult{
		{Job: Job{InputPath: "a.jpg", OutputPath: "a.webp"}, Success: true, Attempts: 1, Duration: time.Second},
		{Job: Job{InputPath: "b.jpg", OutputPath: "b.webp"}, Skipped: true, SkipReason: "output_exists"},
	}

	out, err := RenderReport(ReportTXT, summary, results, time.Unix(0, 0), time.Unix(2, 0))
	if err != nil {
		t.Fatalf("RenderReport failed: %v", err)
	}
	if !strings.Contains(out, "Batch Report") {
		t.Fatalf("missing report header")
	}
	if !strings.Contains(out, "[success] a.jpg -> a.webp") {
		t.Fatalf("missing success item")
	}
	if !strings.Contains(out, "[skipped] b.jpg -> b.webp") {
		t.Fatalf("missing skipped item")
	}
}

func TestRenderReportJSON(t *testing.T) {
	summary := Summary{
		Total:     1,
		Succeeded: 0,
		Skipped:   0,
		Failed:    1,
		Duration:  time.Second,
	}
	results := []JobResult{
		{Job: Job{InputPath: "x", OutputPath: "y"}, Attempts: 3, Error: errStub("boom")},
	}
	out, err := RenderReport(ReportJSON, summary, results, time.Unix(0, 0), time.Unix(1, 0))
	if err != nil {
		t.Fatalf("RenderReport failed: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}

	if payload["total"] != float64(1) {
		t.Fatalf("unexpected total: %v", payload["total"])
	}

	items, ok := payload["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected items: %v", payload["items"])
	}

	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected first item type")
	}
	if first["status"] != "failed" {
		t.Fatalf("unexpected status: %v", first["status"])
	}
}

type errStub string

func (e errStub) Error() string { return string(e) }
