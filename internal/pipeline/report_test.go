package pipeline

import "testing"

func TestNormalizeReportFormat(t *testing.T) {
	if got := NormalizeReportFormat(""); got != ReportOff {
		t.Fatalf("expected off, got %s", got)
	}
	if got := NormalizeReportFormat("JSON"); got != ReportJSON {
		t.Fatalf("expected json, got %s", got)
	}
	if got := NormalizeReportFormat("bad"); got != "" {
		t.Fatalf("expected empty for invalid format, got %s", got)
	}
}

func TestRenderReport(t *testing.T) {
	r := Result{
		Input:       "in.txt",
		FinalOutput: "out.md",
		Steps: []StepResult{
			{Index: 1, Type: "convert", Input: "in.txt", Output: "out.md", Success: true},
		},
	}

	txt, err := RenderReport(ReportTXT, r)
	if err != nil {
		t.Fatalf("RenderReport txt failed: %v", err)
	}
	if txt == "" {
		t.Fatalf("expected txt report")
	}

	js, err := RenderReport(ReportJSON, r)
	if err != nil {
		t.Fatalf("RenderReport json failed: %v", err)
	}
	if js == "" {
		t.Fatalf("expected json report")
	}
}
