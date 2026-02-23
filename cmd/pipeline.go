package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlihgenel/fileconverter-cli/internal/pipeline"
	"github.com/mlihgenel/fileconverter-cli/internal/ui"
)

var (
	pipelineProfile    string
	pipelineQuality    int
	pipelineOnConflict string
	pipelinePreserveMD bool
	pipelineStripMD    bool
	pipelineReport     string
	pipelineReportFile string
	pipelineResumeFile string
	pipelineKeepTemps  bool
)

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Çok adımlı dönüşüm akışları",
}

var pipelineRunCmd = &cobra.Command{
	Use:   "run <pipeline.json>",
	Short: "Pipeline spec dosyasını çalıştır",
	Long: `JSON formatında tanımlanan pipeline akışını sırayla çalıştırır.

Örnek:
  fileconverter-cli pipeline run ./pipeline.json --profile social-story
  fileconverter-cli pipeline run ./pipeline.json --strip-metadata --report json --report-file ./reports/pipeline.json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		specPath := args[0]
		jsonOutput := isJSONOutput()

		applyProfileDefault(cmd, "profile", &pipelineProfile)
		applyQualityDefault(cmd, "quality", &pipelineQuality)
		applyOnConflictDefault(cmd, "on-conflict", &pipelineOnConflict)
		applyMetadataDefault(cmd, "preserve-metadata", &pipelinePreserveMD, "strip-metadata", &pipelineStripMD)
		applyReportDefault(cmd, "report", &pipelineReport)

		if p, ok, err := resolveProfile(pipelineProfile); err != nil {
			ui.PrintError(err.Error())
			return err
		} else if ok {
			applyProfileToPipeline(cmd, p)
			applyProfileMetadata(cmd, p, "preserve-metadata", &pipelinePreserveMD, "strip-metadata", &pipelineStripMD)
		}

		metadataMode, err := metadataModeFromFlags(pipelinePreserveMD, pipelineStripMD)
		if err != nil {
			ui.PrintError(err.Error())
			return err
		}

		conflictPolicy := pipelineOnConflict
		reportFormat := pipeline.NormalizeReportFormat(pipelineReport)
		if reportFormat == "" {
			err := fmt.Errorf("gecersiz report formati: %s", pipelineReport)
			ui.PrintError(err.Error())
			return err
		}

		spec, err := pipeline.LoadSpec(specPath)
		if err != nil {
			ui.PrintError(err.Error())
			return err
		}
		spec = resolvePipelinePaths(spec, specPath)
		resumePlan, err := buildPipelineResumePlan(spec, pipelineResumeFile)
		if err != nil {
			ui.PrintError(err.Error())
			return err
		}

		if !jsonOutput {
			ui.PrintInfo(fmt.Sprintf("Pipeline çalıştırılıyor: %s", specPath))
		}
		if pipelineProfile != "" && !jsonOutput {
			ui.PrintInfo(fmt.Sprintf("Profil: %s", pipelineProfile))
		}
		if strings.TrimSpace(pipelineResumeFile) != "" && !jsonOutput {
			if resumePlan.StepOffset > 0 {
				ui.PrintInfo(fmt.Sprintf("Resume: ilk %d step başarılı bulundu, kaldığı yerden devam ediliyor.", resumePlan.StepOffset))
			} else {
				ui.PrintInfo("Resume: uygun başarılı step bulunamadı, pipeline baştan çalışacak.")
			}
		}

		started := time.Now()
		result := pipeline.Result{}
		var execErr error
		if resumePlan.SkipExecution {
			result = mergePipelineResumeResult(resumePlan, pipeline.Result{}, started)
			if !jsonOutput {
				ui.PrintInfo("Pipeline zaten tamamlanmış görünüyor; yeniden çalıştırma atlandı.")
			}
		} else {
			partial, runErr := pipeline.Execute(resumePlan.RunSpec, pipeline.ExecuteConfig{
				OutputDir:      outputDir,
				Verbose:        verbose,
				DefaultQuality: pipelineQuality,
				MetadataMode:   metadataMode,
				OnConflict:     conflictPolicy,
				KeepTemps:      pipelineKeepTemps,
			})
			execErr = runErr
			result = mergePipelineResumeResult(resumePlan, partial, started)
		}
		elapsed := time.Since(started)

		if !jsonOutput {
			if execErr != nil {
				ui.PrintError(fmt.Sprintf("Pipeline başarısız: %s", execErr.Error()))
			} else {
				ui.PrintSuccess("Pipeline tamamlandı.")
			}
			ui.PrintDuration(elapsed)

			for _, s := range result.Steps {
				if s.Success {
					ui.PrintInfo(fmt.Sprintf("Step %d (%s): %s -> %s", s.Index, s.Type, s.Input, s.Output))
				} else {
					ui.PrintError(fmt.Sprintf("Step %d (%s) hatası: %s", s.Index, s.Type, s.Error))
				}
			}
		}

		reportText, reportErr := pipeline.RenderReport(reportFormat, result)
		if reportErr != nil {
			ui.PrintError(fmt.Sprintf("Rapor üretilemedi: %s", reportErr.Error()))
			if execErr != nil {
				return execErr
			}
			return reportErr
		}
		if strings.TrimSpace(reportText) != "" {
			if strings.TrimSpace(pipelineReportFile) != "" {
				if err := writeBatchReport(pipelineReportFile, reportText); err != nil {
					ui.PrintError(fmt.Sprintf("Rapor yazılamadı: %s", err.Error()))
					if execErr != nil {
						return execErr
					}
					return err
				}
				if !jsonOutput {
					ui.PrintInfo(fmt.Sprintf("Rapor yazıldı: %s", pipelineReportFile))
				}
			} else if !jsonOutput {
				fmt.Println(reportText)
			}
		}
		if jsonOutput {
			jsonReport, err := pipeline.RenderReport(pipeline.ReportJSON, result)
			if err != nil {
				return err
			}
			fmt.Println(jsonReport)
		}

		if execErr != nil {
			return execErr
		}
		return nil
	},
}

func init() {
	pipelineRunCmd.Flags().StringVar(&pipelineProfile, "profile", "", "Hazır profil (ör: social-story, podcast-clean, archive-lossless)")
	pipelineRunCmd.Flags().IntVarP(&pipelineQuality, "quality", "q", 0, "Varsayılan kalite seviyesi (1-100)")
	pipelineRunCmd.Flags().StringVar(&pipelineOnConflict, "on-conflict", "versioned", "Çakışma politikası: overwrite, skip, versioned")
	pipelineRunCmd.Flags().BoolVar(&pipelinePreserveMD, "preserve-metadata", false, "Metadata bilgisini korumayı dene")
	pipelineRunCmd.Flags().BoolVar(&pipelineStripMD, "strip-metadata", false, "Metadata bilgisini temizle")
	pipelineRunCmd.Flags().StringVar(&pipelineReport, "report", pipeline.ReportTXT, "Rapor formatı: off, txt, json")
	pipelineRunCmd.Flags().StringVar(&pipelineReportFile, "report-file", "", "Raporu belirtilen dosyaya yaz")
	pipelineRunCmd.Flags().StringVar(&pipelineResumeFile, "resume-from-report", "", "Önceki JSON rapordan başarılı step'leri okuyup kaldığı yerden devam et")
	pipelineRunCmd.Flags().BoolVar(&pipelineKeepTemps, "keep-temps", false, "Ara geçici dosyaları silme")

	pipelineCmd.AddCommand(pipelineRunCmd)
	rootCmd.AddCommand(pipelineCmd)
}

func resolvePipelinePaths(spec pipeline.Spec, specPath string) pipeline.Spec {
	baseDir := filepath.Dir(specPath)

	resolve := func(path string) string {
		if strings.TrimSpace(path) == "" {
			return path
		}
		if filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(baseDir, path)
	}

	spec.Input = resolve(spec.Input)
	spec.Output = resolve(spec.Output)
	for i := range spec.Steps {
		spec.Steps[i].Output = resolve(spec.Steps[i].Output)
	}
	return spec
}
