package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlihgenel/fileconverter-cli/internal/batch"
	"github.com/mlihgenel/fileconverter-cli/internal/converter"
	"github.com/mlihgenel/fileconverter-cli/internal/ui"
)

var (
	batchTo         string
	batchFrom       string
	batchProfile    string
	batchRecursive  bool
	batchDryRun     bool
	batchQuality    int
	batchOnConflict string
	batchPreserveMD bool
	batchStripMD    bool
	batchRetry      int
	batchRetryDelay time.Duration
	batchReport     string
	batchReportFile string
	batchPreset     string
	batchWidth      float64
	batchHeight     float64
	batchUnit       string
	batchResizeDPI  float64
	batchResizeMode string
)

var batchCmd = &cobra.Command{
	Use:   "batch <dizin veya glob>",
	Short: "Birden fazla dosyayı toplu dönüştür",
	Long: `Bir dizindeki veya glob pattern'e uyan tüm dosyaları toplu olarak dönüştürür.
Worker pool kullanarak paralel dönüşüm yapar.

Örnekler:
  fileconverter-cli batch ./belgeler --from md --to pdf
  fileconverter-cli batch ./belgeler --from md --to pdf --recursive
  fileconverter-cli batch ./muzikler --from mp3 --to wav --workers 8
  fileconverter-cli batch ./videolar --from mp4 --to gif --quality 80
  fileconverter-cli batch "*.png" --to jpg --quality 85
  fileconverter-cli batch ./resimler --from png --to jpg --dry-run
  fileconverter-cli batch ./belgeler --from md --to html --output ./cikti/
  fileconverter-cli batch ./videolar --from mp4 --to mp4 --preset story --resize-mode pad
  fileconverter-cli batch ./fotograflar --from webp --to png --width 10 --height 15 --unit cm --dpi 300
  fileconverter-cli batch ./resimler --from jpg --to png --on-conflict versioned --retry 2 --report json --report-file ./reports/batch.json
  fileconverter-cli batch ./videolar --from mov --to mp4 --profile archive-lossless --preserve-metadata`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]
		applyProfileDefault(cmd, "profile", &batchProfile)
		applyQualityDefault(cmd, "quality", &batchQuality)
		applyOnConflictDefault(cmd, "on-conflict", &batchOnConflict)
		applyMetadataDefault(cmd, "preserve-metadata", &batchPreserveMD, "strip-metadata", &batchStripMD)
		applyRetryDefaults(cmd, "retry", &batchRetry, "retry-delay", &batchRetryDelay)
		applyReportDefault(cmd, "report", &batchReport)

		if p, ok, err := resolveProfile(batchProfile); err != nil {
			ui.PrintError(err.Error())
			return err
		} else if ok {
			applyProfileToBatch(cmd, p)
			applyProfileMetadata(cmd, p, "preserve-metadata", &batchPreserveMD, "strip-metadata", &batchStripMD)
		}

		metadataMode, err := metadataModeFromFlags(batchPreserveMD, batchStripMD)
		if err != nil {
			ui.PrintError(err.Error())
			return err
		}

		conflictPolicy := converter.NormalizeConflictPolicy(batchOnConflict)
		if conflictPolicy == "" {
			err := fmt.Errorf("gecersiz on-conflict politikasi: %s", batchOnConflict)
			ui.PrintError(err.Error())
			return err
		}
		reportFormat := batch.NormalizeReportFormat(batchReport)
		if reportFormat == "" {
			err := fmt.Errorf("gecersiz report formati: %s", batchReport)
			ui.PrintError(err.Error())
			return err
		}

		// Hedef format kontrolü
		targetFormat := converter.NormalizeFormat(batchTo)
		if targetFormat == "" {
			ui.PrintError("Hedef format belirtilmedi. --to <format> kullanın.")
			return fmt.Errorf("hedef format belirtilmedi")
		}

		// Kaynak format kontrolü
		fromFormat := converter.NormalizeFormat(batchFrom)
		if fromFormat == "" {
			ui.PrintError("Kaynak format belirtilmedi. --from <format> kullanın.")
			return fmt.Errorf("kaynak format belirtilmedi")
		}

		resizeSpec, err := converter.BuildResizeSpec(
			batchPreset,
			batchWidth,
			batchHeight,
			batchUnit,
			batchResizeMode,
			batchResizeDPI,
		)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Boyutlandırma parametreleri hatalı: %s", err.Error()))
			return err
		}
		if resizeSpec != nil && !converter.IsResizableFormat(fromFormat) {
			err := fmt.Errorf("boyutlandırma sadece görsel ve video dosyalarında kullanılabilir")
			ui.PrintError(err.Error())
			return err
		}
		// Aynı format, resize yoksa no-op
		if fromFormat == targetFormat && resizeSpec == nil {
			ui.PrintWarning("Kaynak ve hedef format aynı, dönüşüm gerekli değil.")
			return nil
		}

		// Dönüşüm desteği kontrolü
		_, err = converter.FindConverter(fromFormat, targetFormat)
		if err != nil {
			ui.PrintError(err.Error())
			return err
		}

		// Dosyaları topla
		var files []string
		info, statErr := os.Stat(source)
		if statErr == nil && info.IsDir() {
			// Dizin modu
			files, err = batch.CollectFiles(source, fromFormat, batchRecursive)
			if err != nil {
				ui.PrintError(fmt.Sprintf("Dizin taranamadı: %s", err.Error()))
				return err
			}
		} else {
			// Glob pattern modu
			files, err = batch.CollectFilesFromGlob(source)
			if err != nil {
				ui.PrintError(fmt.Sprintf("Glob pattern hatası: %s", err.Error()))
				return err
			}
			// Sadece doğru uzantıya sahip dosyaları filtrele
			var filtered []string
			for _, f := range files {
				if converter.HasFormatExtension(f, fromFormat) {
					filtered = append(filtered, f)
				}
			}
			files = filtered
		}

		if len(files) == 0 {
			ui.PrintWarning(fmt.Sprintf("'%s' formatında dosya bulunamadı.", converter.FormatFilterLabel(fromFormat)))
			return nil
		}

		// Dosya bilgisi
		ui.PrintInfo(fmt.Sprintf("%d adet .%s dosyası bulundu", len(files), converter.FormatFilterLabel(fromFormat)))
		if resizeSpec != nil {
			source := "manuel"
			if resizeSpec.Preset != "" {
				source = "preset: " + resizeSpec.Preset
			}
			ui.PrintInfo(fmt.Sprintf("Boyutlandırma: %dx%d (%s, mod: %s)", resizeSpec.Width, resizeSpec.Height, source, resizeSpec.Mode))
		}

		if verbose {
			for _, f := range files {
				fmt.Printf("  %s %s\n", ui.IconFile, f)
			}
			fmt.Println()
		}

		// İşleri oluştur
		jobs := make([]batch.Job, 0, len(files))
		reserved := make(map[string]struct{}, len(files))
		for _, f := range files {
			baseOutput := converter.BuildOutputPath(f, outputDir, targetFormat, "")
			resolvedOutput, skipReason, err := resolveBatchOutputPath(baseOutput, conflictPolicy, reserved)
			if err != nil {
				ui.PrintError(err.Error())
				return err
			}
			jobs = append(jobs, batch.Job{
				InputPath:  f,
				OutputPath: resolvedOutput,
				From:       fromFormat,
				To:         targetFormat,
				SkipReason: skipReason,
				Options: converter.Options{
					Quality:      batchQuality,
					Verbose:      verbose,
					Resize:       resizeSpec,
					MetadataMode: metadataMode,
				},
			})
		}

		// Dry-run modu
		if batchDryRun {
			ui.PrintInfo("Ön izleme modu (--dry-run) — dönüşüm yapılmayacak:")
			fmt.Println()
			skipped := 0
			for _, job := range jobs {
				if job.SkipReason != "" {
					skipped++
					ui.PrintWarning(fmt.Sprintf("Atlanacak: %s (sebep: %s)", job.InputPath, job.SkipReason))
					continue
				}
				ui.PrintConversion(job.InputPath, job.OutputPath)
			}
			fmt.Println()
			ui.PrintInfo(fmt.Sprintf("Toplam %d dosya işlenecek (%d atlanacak).", len(jobs), skipped))
			ui.PrintInfo("Dönüşümü başlatmak için --dry-run flag'ini kaldırın.")
			return nil
		}

		// Worker pool oluştur
		pool := batch.NewPool(workers)
		pool.SetRetry(batchRetry, batchRetryDelay)

		// Progress bar
		pb := ui.NewProgressBar(len(jobs), "Dönüştürülüyor")
		pool.OnProgress = func(completed, total int) {
			pb.Update(completed)
		}

		// Çalıştır
		fmt.Println()
		startedAt := time.Now()
		results := pool.Execute(jobs)
		endedAt := time.Now()
		totalDuration := endedAt.Sub(startedAt)

		// Sonuçları özetle
		summary := batch.GetSummary(results, totalDuration)
		ui.PrintBatchSummary(summary.Total, summary.Succeeded, summary.Skipped, summary.Failed, totalDuration)

		// Hataları göster
		if len(summary.Errors) > 0 {
			ui.PrintError("Başarısız dönüşümler:")
			for _, e := range summary.Errors {
				fmt.Printf("  %s %s: %s (deneme: %d)\n", ui.IconError, e.InputFile, e.Error, e.Attempts)
			}
			fmt.Println()
		}

		reportText, err := batch.RenderReport(reportFormat, summary, results, startedAt, endedAt)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Rapor oluşturulamadı: %s", err.Error()))
			return err
		}
		if reportText != "" {
			if strings.TrimSpace(batchReportFile) != "" {
				if err := writeBatchReport(batchReportFile, reportText); err != nil {
					ui.PrintError(fmt.Sprintf("Rapor dosyaya yazılamadı: %s", err.Error()))
					return err
				}
				ui.PrintInfo(fmt.Sprintf("Rapor yazıldı: %s", batchReportFile))
			} else {
				fmt.Println(reportText)
			}
		}

		if summary.Failed > 0 {
			return fmt.Errorf("%d dosya dönüştürülemedi", summary.Failed)
		}

		return nil
	},
}

func init() {
	batchCmd.Flags().StringVarP(&batchTo, "to", "t", "", "Hedef format (zorunlu)")
	batchCmd.Flags().StringVarP(&batchFrom, "from", "f", "", "Kaynak format (zorunlu)")
	batchCmd.Flags().StringVar(&batchProfile, "profile", "", "Hazır profil (ör: social-story, podcast-clean, archive-lossless)")
	batchCmd.Flags().BoolVarP(&batchRecursive, "recursive", "r", false, "Alt dizinleri de tara")
	batchCmd.Flags().BoolVar(&batchDryRun, "dry-run", false, "Ön izleme — dönüşüm yapmadan listele")
	batchCmd.Flags().IntVarP(&batchQuality, "quality", "q", 0, "Kalite seviyesi (1-100)")
	batchCmd.Flags().StringVar(&batchOnConflict, "on-conflict", converter.ConflictVersioned, "Çakışma politikası: overwrite, skip, versioned")
	batchCmd.Flags().BoolVar(&batchPreserveMD, "preserve-metadata", false, "Metadata bilgisini korumayı dene")
	batchCmd.Flags().BoolVar(&batchStripMD, "strip-metadata", false, "Metadata bilgisini temizle")
	batchCmd.Flags().IntVar(&batchRetry, "retry", 0, "Başarısız işler için otomatik tekrar sayısı")
	batchCmd.Flags().DurationVar(&batchRetryDelay, "retry-delay", 500*time.Millisecond, "Retry denemeleri arası bekleme (örn: 500ms, 2s)")
	batchCmd.Flags().StringVar(&batchReport, "report", batch.ReportOff, "Rapor formatı: off, txt, json")
	batchCmd.Flags().StringVar(&batchReportFile, "report-file", "", "Raporu belirtilen dosyaya yaz")
	batchCmd.Flags().StringVar(&batchPreset, "preset", "", "Hazır boyut preset'i (ör: story, square, fullhd, 1080x1920)")
	batchCmd.Flags().Float64Var(&batchWidth, "width", 0, "Manuel hedef genişlik")
	batchCmd.Flags().Float64Var(&batchHeight, "height", 0, "Manuel hedef yükseklik")
	batchCmd.Flags().StringVar(&batchUnit, "unit", "px", "Manuel ölçü birimi: px veya cm")
	batchCmd.Flags().Float64Var(&batchResizeDPI, "dpi", 96, "Birim cm ise kullanılacak DPI değeri")
	batchCmd.Flags().StringVar(&batchResizeMode, "resize-mode", "pad", "Boyutlandırma modu: pad, fit, fill, stretch")

	batchCmd.MarkFlagRequired("to")
	batchCmd.MarkFlagRequired("from")

	rootCmd.AddCommand(batchCmd)
}

func resolveBatchOutputPath(baseOutput, conflictPolicy string, reserved map[string]struct{}) (string, string, error) {
	exists := func(path string) bool {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		return false
	}

	_, alreadyReserved := reserved[baseOutput]
	if conflictPolicy == converter.ConflictOverwrite {
		reserved[baseOutput] = struct{}{}
		return baseOutput, "", nil
	}

	if conflictPolicy == converter.ConflictSkip {
		if exists(baseOutput) || alreadyReserved {
			return baseOutput, "output_exists", nil
		}
		reserved[baseOutput] = struct{}{}
		return baseOutput, "", nil
	}

	if !exists(baseOutput) && !alreadyReserved {
		reserved[baseOutput] = struct{}{}
		return baseOutput, "", nil
	}

	ext := filepath.Ext(baseOutput)
	base := strings.TrimSuffix(baseOutput, ext)
	for i := 1; i < 100000; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if _, used := reserved[candidate]; used {
			continue
		}
		if exists(candidate) {
			continue
		}
		reserved[candidate] = struct{}{}
		return candidate, "", nil
	}

	return "", "", fmt.Errorf("uygun cikti dosya adi bulunamadi: %s", baseOutput)
}

func writeBatchReport(path, content string) error {
	reportDir := filepath.Dir(path)
	if reportDir != "" && reportDir != "." {
		if err := os.MkdirAll(reportDir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(content), 0644)
}
