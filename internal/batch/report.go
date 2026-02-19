package batch

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	ReportOff  = "off"
	ReportTXT  = "txt"
	ReportJSON = "json"
)

type reportItem struct {
	Input      string `json:"input"`
	Output     string `json:"output"`
	Status     string `json:"status"`
	Attempts   int    `json:"attempts,omitempty"`
	DurationMS int64  `json:"duration_ms"`
	OutputSize int64  `json:"output_size,omitempty"`
	Error      string `json:"error,omitempty"`
	SkipReason string `json:"skip_reason,omitempty"`
}

type reportPayload struct {
	StartedAt string       `json:"started_at"`
	EndedAt   string       `json:"ended_at"`
	Duration  string       `json:"duration"`
	Total     int          `json:"total"`
	Succeeded int          `json:"succeeded"`
	Skipped   int          `json:"skipped"`
	Failed    int          `json:"failed"`
	Items     []reportItem `json:"items"`
}

// NormalizeReportFormat rapor formatını normalize eder.
func NormalizeReportFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", ReportOff:
		return ReportOff
	case ReportTXT:
		return ReportTXT
	case ReportJSON:
		return ReportJSON
	default:
		return ""
	}
}

// RenderReport batch sonucu için rapor metni üretir.
func RenderReport(format string, summary Summary, results []JobResult, startedAt, endedAt time.Time) (string, error) {
	switch NormalizeReportFormat(format) {
	case ReportOff:
		return "", nil
	case ReportTXT:
		return renderTXTReport(summary, results, startedAt, endedAt), nil
	case ReportJSON:
		return renderJSONReport(summary, results, startedAt, endedAt)
	default:
		return "", fmt.Errorf("gecersiz report formati: %s", format)
	}
}

func renderTXTReport(summary Summary, results []JobResult, startedAt, endedAt time.Time) string {
	var b strings.Builder
	b.WriteString("Batch Report\n")
	b.WriteString(strings.Repeat("=", 40))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Started:   %s\n", startedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("Ended:     %s\n", endedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("Duration:  %s\n", summary.Duration))
	b.WriteString(fmt.Sprintf("Total:     %d\n", summary.Total))
	b.WriteString(fmt.Sprintf("Succeeded: %d\n", summary.Succeeded))
	b.WriteString(fmt.Sprintf("Skipped:   %d\n", summary.Skipped))
	b.WriteString(fmt.Sprintf("Failed:    %d\n", summary.Failed))
	b.WriteString("\nItems:\n")

	for _, r := range results {
		status := "failed"
		switch {
		case r.Success:
			status = "success"
		case r.Skipped:
			status = "skipped"
		}

		b.WriteString(fmt.Sprintf("- [%s] %s -> %s", status, r.Job.InputPath, r.Job.OutputPath))
		if r.Attempts > 0 {
			b.WriteString(fmt.Sprintf(" (attempts=%d)", r.Attempts))
		}
		if r.OutputSize > 0 {
			b.WriteString(fmt.Sprintf(" (size=%d)", r.OutputSize))
		}
		if r.Skipped && r.SkipReason != "" {
			b.WriteString(fmt.Sprintf(" (reason=%s)", r.SkipReason))
		}
		if r.Error != nil {
			b.WriteString(fmt.Sprintf(" (error=%s)", r.Error.Error()))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func renderJSONReport(summary Summary, results []JobResult, startedAt, endedAt time.Time) (string, error) {
	items := make([]reportItem, 0, len(results))
	for _, r := range results {
		item := reportItem{
			Input:      r.Job.InputPath,
			Output:     r.Job.OutputPath,
			Attempts:   r.Attempts,
			DurationMS: r.Duration.Milliseconds(),
			OutputSize: r.OutputSize,
		}

		switch {
		case r.Success:
			item.Status = "success"
		case r.Skipped:
			item.Status = "skipped"
			item.SkipReason = r.SkipReason
		default:
			item.Status = "failed"
			if r.Error != nil {
				item.Error = r.Error.Error()
			}
		}

		items = append(items, item)
	}

	payload := reportPayload{
		StartedAt: startedAt.Format(time.RFC3339),
		EndedAt:   endedAt.Format(time.RFC3339),
		Duration:  summary.Duration.String(),
		Total:     summary.Total,
		Succeeded: summary.Succeeded,
		Skipped:   summary.Skipped,
		Failed:    summary.Failed,
		Items:     items,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
