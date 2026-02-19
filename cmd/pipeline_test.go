package cmd

import (
	"path/filepath"
	"testing"

	"github.com/mlihgenel/fileconverter-cli/internal/pipeline"
)

func TestResolvePipelinePaths(t *testing.T) {
	specPath := filepath.Join("/tmp", "project", "pipeline.json")
	spec := pipeline.Spec{
		Input:  "in.txt",
		Output: "out.md",
		Steps: []pipeline.Step{
			{Type: "convert", To: "md", Output: "step1.md"},
		},
	}

	got := resolvePipelinePaths(spec, specPath)
	base := filepath.Join("/tmp", "project")
	if got.Input != filepath.Join(base, "in.txt") {
		t.Fatalf("unexpected input path: %s", got.Input)
	}
	if got.Output != filepath.Join(base, "out.md") {
		t.Fatalf("unexpected output path: %s", got.Output)
	}
	if got.Steps[0].Output != filepath.Join(base, "step1.md") {
		t.Fatalf("unexpected step output path: %s", got.Steps[0].Output)
	}
}
