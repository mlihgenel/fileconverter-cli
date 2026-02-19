package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	ReportOff  = "off"
	ReportTXT  = "txt"
	ReportJSON = "json"
)

// NormalizeReportFormat formatı normalize eder.
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

// RenderReport pipeline sonucu için rapor üretir.
func RenderReport(format string, result Result) (string, error) {
	switch NormalizeReportFormat(format) {
	case ReportOff:
		return "", nil
	case ReportTXT:
		return renderTXT(result), nil
	case ReportJSON:
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("gecersiz report formati: %s", format)
	}
}

func renderTXT(result Result) string {
	var b strings.Builder
	b.WriteString("Pipeline Report\n")
	b.WriteString(strings.Repeat("=", 40))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Input:      %s\n", result.Input))
	b.WriteString(fmt.Sprintf("Final:      %s\n", result.FinalOutput))
	b.WriteString(fmt.Sprintf("Duration:   %s\n", result.Duration))
	b.WriteString(fmt.Sprintf("Step Count: %d\n", len(result.Steps)))
	b.WriteString("\nSteps:\n")
	for _, s := range result.Steps {
		status := "ok"
		if !s.Success {
			status = "failed"
		}
		b.WriteString(fmt.Sprintf("- [%s] #%d %s: %s -> %s (%s)", status, s.Index, s.Type, s.Input, s.Output, s.Duration))
		if s.Error != "" {
			b.WriteString(fmt.Sprintf(" error=%s", s.Error))
		}
		b.WriteString("\n")
	}
	return b.String()
}
