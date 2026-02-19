package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSpec(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pipeline.json")
	content := `{
  "input": "in.txt",
  "steps": [
    {"type":"convert","to":"md"}
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	spec, err := LoadSpec(path)
	if err != nil {
		t.Fatalf("LoadSpec failed: %v", err)
	}
	if spec.Input != "in.txt" {
		t.Fatalf("unexpected input: %s", spec.Input)
	}
	if len(spec.Steps) != 1 || spec.Steps[0].Type != "convert" {
		t.Fatalf("unexpected steps: %#v", spec.Steps)
	}
}

func TestValidateSpecErrors(t *testing.T) {
	if err := ValidateSpec(Spec{}); err == nil {
		t.Fatalf("expected error for empty spec")
	}

	err := ValidateSpec(Spec{
		Input: "in.txt",
		Steps: []Step{{Type: "convert"}},
	})
	if err == nil {
		t.Fatalf("expected error for convert without to")
	}
}
