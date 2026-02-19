package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	// converter kayıtları için side-effect import
	_ "github.com/mlihgenel/fileconverter-cli/internal/converter"
)

func TestExecuteConvertPipeline(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(input, []byte("hello pipeline"), 0644); err != nil {
		t.Fatalf("write input failed: %v", err)
	}

	spec := Spec{
		Input: input,
		Steps: []Step{
			{Type: StepConvert, To: "md"},
		},
	}

	result, err := Execute(spec, ExecuteConfig{
		OutputDir:      dir,
		DefaultQuality: 80,
		MetadataMode:   "auto",
		OnConflict:     "versioned",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(result.Steps) != 1 {
		t.Fatalf("unexpected step count: %d", len(result.Steps))
	}
	if !result.Steps[0].Success {
		t.Fatalf("step should be success: %#v", result.Steps[0])
	}
	if result.FinalOutput == "" {
		t.Fatalf("final output should not be empty")
	}
	if _, err := os.Stat(result.FinalOutput); err != nil {
		t.Fatalf("final output not found: %v", err)
	}
}

func TestExecuteInvalidSpec(t *testing.T) {
	_, err := Execute(Spec{}, ExecuteConfig{})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}
