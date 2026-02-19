package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	StepConvert        = "convert"
	StepAudioNormalize = "audio-normalize"
)

// Spec pipeline tanımını temsil eder.
type Spec struct {
	Input string `json:"input"`
	// Output son adım için nihai çıktı dosya yolunu zorlar.
	Output string `json:"output,omitempty"`
	Steps  []Step `json:"steps"`
}

// Step pipeline içindeki tek bir adımı temsil eder.
type Step struct {
	Type string `json:"type"`

	// convert
	To      string `json:"to,omitempty"`
	Quality int    `json:"quality,omitempty"`

	// Ortak
	Output       string `json:"output,omitempty"`
	MetadataMode string `json:"metadata_mode,omitempty"`

	// audio-normalize
	TargetLUFS float64 `json:"target_lufs,omitempty"`
	TargetTP   float64 `json:"target_tp,omitempty"`
	TargetLRA  float64 `json:"target_lra,omitempty"`
}

// LoadSpec JSON spec dosyasını yükler.
func LoadSpec(path string) (Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Spec{}, err
	}

	var s Spec
	if err := json.Unmarshal(data, &s); err != nil {
		return Spec{}, fmt.Errorf("pipeline spec parse hatasi: %w", err)
	}

	if err := ValidateSpec(s); err != nil {
		return Spec{}, err
	}

	return s, nil
}

// ValidateSpec pipeline spec doğrulaması yapar.
func ValidateSpec(s Spec) error {
	if strings.TrimSpace(s.Input) == "" {
		return fmt.Errorf("input zorunlu")
	}
	if len(s.Steps) == 0 {
		return fmt.Errorf("en az bir step gerekli")
	}

	for i, step := range s.Steps {
		t := strings.ToLower(strings.TrimSpace(step.Type))
		if t == "" {
			return fmt.Errorf("step[%d] type zorunlu", i)
		}
		switch t {
		case StepConvert:
			if strings.TrimSpace(step.To) == "" {
				return fmt.Errorf("step[%d] convert icin to zorunlu", i)
			}
		case StepAudioNormalize:
			// opsiyonel alanlar runtime'da defaultlanır.
		default:
			return fmt.Errorf("step[%d] desteklenmeyen type: %s", i, step.Type)
		}
	}

	return nil
}
