package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mlihgenel/fileconverter-cli/internal/pipeline"
)

func TestLoadBatchResumeSuccess(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "batch.json")
	report := map[string]any{
		"items": []map[string]any{
			{"input": filepath.Join(dir, "ok.jpg"), "status": "success"},
			{"input": filepath.Join(dir, "bad.jpg"), "status": "failed"},
		},
	}
	data, _ := json.Marshal(report)
	if err := os.WriteFile(reportPath, data, 0644); err != nil {
		t.Fatalf("write report failed: %v", err)
	}

	set, err := loadBatchResumeSuccess(reportPath)
	if err != nil {
		t.Fatalf("loadBatchResumeSuccess failed: %v", err)
	}
	if !hasResumeSuccess(set, filepath.Join(dir, "ok.jpg")) {
		t.Fatalf("expected ok.jpg in success set")
	}
	if hasResumeSuccess(set, filepath.Join(dir, "bad.jpg")) {
		t.Fatalf("did not expect bad.jpg in success set")
	}
}

func TestBuildPipelineResumePlan(t *testing.T) {
	dir := t.TempDir()
	step1Out := filepath.Join(dir, "step1.md")
	step2Out := filepath.Join(dir, "step2.pdf")
	if err := os.WriteFile(step1Out, []byte("s1"), 0644); err != nil {
		t.Fatalf("write step1 output failed: %v", err)
	}
	if err := os.WriteFile(step2Out, []byte("s2"), 0644); err != nil {
		t.Fatalf("write step2 output failed: %v", err)
	}

	spec := pipeline.Spec{
		Input: filepath.Join(dir, "in.txt"),
		Steps: []pipeline.Step{
			{Type: pipeline.StepConvert, To: "md"},
			{Type: pipeline.StepConvert, To: "pdf"},
			{Type: pipeline.StepConvert, To: "txt"},
		},
	}
	prev := pipeline.Result{
		Steps: []pipeline.StepResult{
			{Index: 1, Type: pipeline.StepConvert, Success: true, Output: step1Out},
			{Index: 2, Type: pipeline.StepConvert, Success: true, Output: step2Out},
		},
	}

	reportPath := filepath.Join(dir, "pipeline.json")
	data, _ := json.Marshal(prev)
	if err := os.WriteFile(reportPath, data, 0644); err != nil {
		t.Fatalf("write report failed: %v", err)
	}

	plan, err := buildPipelineResumePlan(spec, reportPath)
	if err != nil {
		t.Fatalf("buildPipelineResumePlan failed: %v", err)
	}
	if plan.StepOffset != 2 {
		t.Fatalf("expected step offset 2, got %d", plan.StepOffset)
	}
	if plan.RunSpec.Input != step2Out {
		t.Fatalf("expected resume input %s, got %s", step2Out, plan.RunSpec.Input)
	}
	if len(plan.RunSpec.Steps) != 1 {
		t.Fatalf("expected 1 remaining step, got %d", len(plan.RunSpec.Steps))
	}
}

func TestBuildPipelineResumePlanCompleted(t *testing.T) {
	dir := t.TempDir()
	step1Out := filepath.Join(dir, "step1.md")
	step2Out := filepath.Join(dir, "step2.pdf")
	if err := os.WriteFile(step2Out, []byte("s2"), 0644); err != nil {
		t.Fatalf("write step2 output failed: %v", err)
	}

	spec := pipeline.Spec{
		Input: filepath.Join(dir, "in.txt"),
		Steps: []pipeline.Step{
			{Type: pipeline.StepConvert, To: "md"},
			{Type: pipeline.StepConvert, To: "pdf"},
		},
	}
	prev := pipeline.Result{
		EndedAt: time.Now(),
		Steps: []pipeline.StepResult{
			{Index: 1, Type: pipeline.StepConvert, Success: true, Output: step1Out},
			{Index: 2, Type: pipeline.StepConvert, Success: true, Output: step2Out},
		},
	}

	reportPath := filepath.Join(dir, "pipeline.json")
	data, _ := json.Marshal(prev)
	if err := os.WriteFile(reportPath, data, 0644); err != nil {
		t.Fatalf("write report failed: %v", err)
	}

	plan, err := buildPipelineResumePlan(spec, reportPath)
	if err != nil {
		t.Fatalf("buildPipelineResumePlan failed: %v", err)
	}
	if !plan.SkipExecution {
		t.Fatalf("expected skip execution for completed pipeline")
	}
}

func TestMergePipelineResumeResult(t *testing.T) {
	plan := pipelineResumePlan{
		OriginalInput: "/tmp/in.txt",
		StepOffset:    2,
		PreviousSteps: []pipeline.StepResult{
			{Index: 1, Type: "convert", Success: true, Output: "/tmp/1"},
			{Index: 2, Type: "convert", Success: true, Output: "/tmp/2"},
		},
	}
	partial := pipeline.Result{
		FinalOutput: "/tmp/3",
		Steps: []pipeline.StepResult{
			{Index: 1, Type: "convert", Success: true, Output: "/tmp/3"},
		},
	}

	merged := mergePipelineResumeResult(plan, partial, time.Now())
	if len(merged.Steps) != 3 {
		t.Fatalf("expected 3 merged steps, got %d", len(merged.Steps))
	}
	if merged.Steps[2].Index != 3 {
		t.Fatalf("expected adjusted index 3, got %d", merged.Steps[2].Index)
	}
	if merged.FinalOutput != "/tmp/3" {
		t.Fatalf("expected final output /tmp/3, got %s", merged.FinalOutput)
	}
}
