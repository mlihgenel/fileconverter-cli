package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mlihgenel/fileconverter-cli/internal/pipeline"
)

type batchResumePayload struct {
	Items []struct {
		Input  string `json:"input"`
		Status string `json:"status"`
	} `json:"items"`
}

func loadBatchResumeSuccess(reportPath string) (map[string]struct{}, error) {
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, err
	}

	var report batchResumePayload
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("resume raporu JSON parse hatasi: %w", err)
	}

	succeeded := make(map[string]struct{}, len(report.Items))
	for _, item := range report.Items {
		if strings.EqualFold(strings.TrimSpace(item.Status), "success") {
			for _, key := range pathCandidates(item.Input) {
				succeeded[key] = struct{}{}
			}
		}
	}
	return succeeded, nil
}

func hasResumeSuccess(set map[string]struct{}, path string) bool {
	if len(set) == 0 {
		return false
	}
	for _, key := range pathCandidates(path) {
		if _, ok := set[key]; ok {
			return true
		}
	}
	return false
}

func pathCandidates(path string) []string {
	p := strings.TrimSpace(path)
	if p == "" {
		return nil
	}
	clean := filepath.Clean(p)
	candidates := []string{clean}
	if abs, err := filepath.Abs(clean); err == nil {
		candidates = append(candidates, filepath.Clean(abs))
	}
	return candidates
}

type pipelineResumePlan struct {
	OriginalInput string
	RunSpec       pipeline.Spec
	PreviousSteps []pipeline.StepResult
	StepOffset    int
	SkipExecution bool
	CompletedAt   time.Time
}

func buildPipelineResumePlan(spec pipeline.Spec, reportPath string) (pipelineResumePlan, error) {
	plan := pipelineResumePlan{
		OriginalInput: spec.Input,
		RunSpec:       spec,
	}
	if strings.TrimSpace(reportPath) == "" {
		return plan, nil
	}

	data, err := os.ReadFile(reportPath)
	if err != nil {
		return plan, err
	}

	var previous pipeline.Result
	if err := json.Unmarshal(data, &previous); err != nil {
		return plan, fmt.Errorf("resume raporu JSON parse hatasi: %w", err)
	}
	if len(previous.Steps) == 0 {
		return plan, nil
	}
	if len(previous.Steps) > len(spec.Steps) {
		return plan, fmt.Errorf("resume raporundaki step sayisi mevcut pipeline'dan fazla")
	}

	offset := 0
	for i := 0; i < len(previous.Steps) && i < len(spec.Steps); i++ {
		prevStep := previous.Steps[i]
		if !prevStep.Success {
			break
		}
		prevType := strings.ToLower(strings.TrimSpace(prevStep.Type))
		specType := strings.ToLower(strings.TrimSpace(spec.Steps[i].Type))
		if prevType != specType {
			return plan, fmt.Errorf("resume uyumsuz: step[%d] tipi raporda %q, spec'te %q", i+1, prevType, specType)
		}
		offset++
	}
	if offset == 0 {
		return plan, nil
	}

	plan.StepOffset = offset
	plan.PreviousSteps = append([]pipeline.StepResult(nil), previous.Steps[:offset]...)
	plan.CompletedAt = previous.EndedAt

	if offset >= len(spec.Steps) {
		plan.SkipExecution = true
		return plan, nil
	}

	resumeInput := strings.TrimSpace(previous.Steps[offset-1].Output)
	if resumeInput == "" {
		return plan, fmt.Errorf("resume için step[%d] output bilgisi boş", offset)
	}
	if _, err := os.Stat(resumeInput); err != nil {
		return plan, fmt.Errorf("resume girdisi bulunamadi: %s", resumeInput)
	}

	plan.RunSpec.Input = resumeInput
	plan.RunSpec.Steps = append([]pipeline.Step(nil), spec.Steps[offset:]...)
	return plan, nil
}

func mergePipelineResumeResult(plan pipelineResumePlan, partial pipeline.Result, started time.Time) pipeline.Result {
	merged := pipeline.Result{
		Input:     plan.OriginalInput,
		StartedAt: started,
		EndedAt:   time.Now(),
	}

	merged.Steps = append(merged.Steps, plan.PreviousSteps...)
	for _, step := range partial.Steps {
		step.Index += plan.StepOffset
		merged.Steps = append(merged.Steps, step)
	}

	switch {
	case partial.FinalOutput != "":
		merged.FinalOutput = partial.FinalOutput
	case len(merged.Steps) > 0:
		merged.FinalOutput = merged.Steps[len(merged.Steps)-1].Output
	}

	merged.Duration = time.Since(started)
	return merged
}
