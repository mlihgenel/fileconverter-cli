package pipeline

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mlihgenel/fileconverter-cli/internal/converter"
)

// ExecuteConfig pipeline çalışma ayarlarını tutar.
type ExecuteConfig struct {
	OutputDir      string
	Verbose        bool
	DefaultQuality int
	MetadataMode   string
	OnConflict     string
	KeepTemps      bool
}

// Result pipeline çalıştırma sonucunu tutar.
type Result struct {
	Input       string
	FinalOutput string
	StartedAt   time.Time
	EndedAt     time.Time
	Duration    time.Duration
	Steps       []StepResult
}

// StepResult tek bir step'in sonucunu tutar.
type StepResult struct {
	Index    int
	Type     string
	Input    string
	Output   string
	Duration time.Duration
	Success  bool
	Error    string
}

// Execute spec'i sırayla çalıştırır.
func Execute(spec Spec, cfg ExecuteConfig) (Result, error) {
	if err := ValidateSpec(spec); err != nil {
		return Result{}, err
	}

	conflict := converter.NormalizeConflictPolicy(cfg.OnConflict)
	if conflict == "" {
		conflict = converter.ConflictVersioned
	}
	metadataMode := converter.NormalizeMetadataMode(cfg.MetadataMode)
	if metadataMode == "" {
		metadataMode = converter.MetadataAuto
	}

	startedAt := time.Now()
	result := Result{
		Input:     spec.Input,
		StartedAt: startedAt,
		Steps:     make([]StepResult, 0, len(spec.Steps)),
	}

	tempDir, err := os.MkdirTemp("", "fileconverter-pipeline-*")
	if err != nil {
		return Result{}, err
	}
	if !cfg.KeepTemps {
		defer os.RemoveAll(tempDir)
	}

	currentInput := spec.Input
	for i, step := range spec.Steps {
		stepStart := time.Now()
		stepType := strings.ToLower(strings.TrimSpace(step.Type))
		var output string

		switch stepType {
		case StepConvert:
			to := converter.NormalizeFormat(step.To)
			output, err = buildStepOutput(currentInput, i, to, step, spec, cfg.OutputDir, tempDir, conflict, len(spec.Steps))
			if err != nil {
				sr := StepResult{
					Index:    i + 1,
					Type:     stepType,
					Input:    currentInput,
					Output:   output,
					Duration: time.Since(stepStart),
					Success:  false,
					Error:    err.Error(),
				}
				result.Steps = append(result.Steps, sr)
				result.EndedAt = time.Now()
				result.Duration = result.EndedAt.Sub(result.StartedAt)
				return result, err
			}

			from := converter.DetectFormat(currentInput)
			conv, err := converter.FindConverter(from, to)
			if err != nil {
				sr := StepResult{
					Index:    i + 1,
					Type:     stepType,
					Input:    currentInput,
					Output:   output,
					Duration: time.Since(stepStart),
					Success:  false,
					Error:    err.Error(),
				}
				result.Steps = append(result.Steps, sr)
				result.EndedAt = time.Now()
				result.Duration = result.EndedAt.Sub(result.StartedAt)
				return result, err
			}

			quality := cfg.DefaultQuality
			if step.Quality > 0 {
				quality = step.Quality
			}
			stepMetadata := metadataMode
			if m := converter.NormalizeMetadataMode(step.MetadataMode); m != "" {
				stepMetadata = m
			}
			opts := converter.Options{
				Quality:      quality,
				Verbose:      cfg.Verbose,
				MetadataMode: stepMetadata,
			}
			err = conv.Convert(currentInput, output, opts)
			if err != nil {
				sr := StepResult{
					Index:    i + 1,
					Type:     stepType,
					Input:    currentInput,
					Output:   output,
					Duration: time.Since(stepStart),
					Success:  false,
					Error:    err.Error(),
				}
				result.Steps = append(result.Steps, sr)
				result.EndedAt = time.Now()
				result.Duration = result.EndedAt.Sub(result.StartedAt)
				return result, err
			}

		case StepAudioNormalize:
			audioOut := converter.DetectFormat(currentInput)
			if step.To != "" {
				audioOut = converter.NormalizeFormat(step.To)
			}
			output, err = buildStepOutput(currentInput, i, audioOut, step, spec, cfg.OutputDir, tempDir, conflict, len(spec.Steps))
			if err != nil {
				sr := StepResult{
					Index:    i + 1,
					Type:     stepType,
					Input:    currentInput,
					Output:   output,
					Duration: time.Since(stepStart),
					Success:  false,
					Error:    err.Error(),
				}
				result.Steps = append(result.Steps, sr)
				result.EndedAt = time.Now()
				result.Duration = result.EndedAt.Sub(result.StartedAt)
				return result, err
			}
			err = runAudioNormalize(currentInput, output, step, metadataMode, cfg.Verbose)
			if err != nil {
				sr := StepResult{
					Index:    i + 1,
					Type:     stepType,
					Input:    currentInput,
					Output:   output,
					Duration: time.Since(stepStart),
					Success:  false,
					Error:    err.Error(),
				}
				result.Steps = append(result.Steps, sr)
				result.EndedAt = time.Now()
				result.Duration = result.EndedAt.Sub(result.StartedAt)
				return result, err
			}
		}

		sr := StepResult{
			Index:    i + 1,
			Type:     stepType,
			Input:    currentInput,
			Output:   output,
			Duration: time.Since(stepStart),
			Success:  true,
		}
		result.Steps = append(result.Steps, sr)
		currentInput = output
	}

	result.FinalOutput = currentInput
	result.EndedAt = time.Now()
	result.Duration = result.EndedAt.Sub(result.StartedAt)
	return result, nil
}

func buildStepOutput(currentInput string, stepIndex int, to string, step Step, spec Spec, outputDir string, tempDir string, conflict string, totalSteps int) (string, error) {
	if strings.TrimSpace(step.Output) != "" {
		return step.Output, nil
	}

	isLast := stepIndex == totalSteps-1
	if isLast && strings.TrimSpace(spec.Output) != "" {
		return spec.Output, nil
	}

	if isLast {
		out := converter.BuildOutputPath(currentInput, outputDir, to, "")
		resolved, skip, err := converter.ResolveOutputPathConflict(out, conflict)
		if err != nil {
			return "", err
		}
		if skip {
			return "", fmt.Errorf("pipeline final output mevcut ve policy skip: %s", out)
		}
		return resolved, nil
	}

	base := strings.TrimSuffix(filepath.Base(currentInput), filepath.Ext(currentInput))
	if base == "" {
		base = "step"
	}
	filename := fmt.Sprintf("%s-step-%d.%s", base, stepIndex+1, to)
	return filepath.Join(tempDir, filename), nil
}

func runAudioNormalize(input string, output string, step Step, defaultMetadataMode string, verbose bool) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("audio-normalize için ffmpeg gerekli")
	}

	targetLUFS := step.TargetLUFS
	if targetLUFS == 0 {
		targetLUFS = -14
	}
	targetTP := step.TargetTP
	if targetTP == 0 {
		targetTP = -1.5
	}
	targetLRA := step.TargetLRA
	if targetLRA == 0 {
		targetLRA = 11
	}

	metadataMode := defaultMetadataMode
	if m := converter.NormalizeMetadataMode(step.MetadataMode); m != "" {
		metadataMode = m
	}

	args := []string{"-i", input, "-y"}
	if !verbose {
		args = append([]string{"-loglevel", "error"}, args...)
	}
	filter := fmt.Sprintf("loudnorm=I=%.1f:TP=%.1f:LRA=%.1f", targetLUFS, targetTP, targetLRA)
	args = append(args, "-af", filter)
	args = append(args, audioCodecArgs(converter.DetectFormat(output))...)
	args = append(args, converter.MetadataFFmpegArgs(metadataMode)...)
	args = append(args, output)

	cmd := exec.Command(ffmpegPath, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("audio-normalize ffmpeg hatasi: %s\n%s", err.Error(), string(out))
	}
	return nil
}

func audioCodecArgs(to string) []string {
	switch converter.NormalizeFormat(to) {
	case "mp3":
		return []string{"-codec:a", "libmp3lame", "-b:a", "192k"}
	case "wav":
		return []string{"-codec:a", "pcm_s16le"}
	case "ogg":
		return []string{"-codec:a", "libvorbis", "-b:a", "192k"}
	case "flac":
		return []string{"-codec:a", "flac"}
	case "aac", "m4a":
		return []string{"-codec:a", "aac", "-b:a", "192k"}
	case "wma":
		return []string{"-codec:a", "wmav2", "-b:a", "192k"}
	case "opus", "webm":
		return []string{"-codec:a", "libopus", "-b:a", "192k"}
	default:
		return []string{"-b:a", "192k"}
	}
}
