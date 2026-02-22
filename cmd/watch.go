package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlihgenel/fileconverter-cli/internal/batch"
	"github.com/mlihgenel/fileconverter-cli/internal/converter"
	"github.com/mlihgenel/fileconverter-cli/internal/ui"
	convwatch "github.com/mlihgenel/fileconverter-cli/internal/watch"
)

var (
	watchTo         string
	watchFrom       string
	watchProfile    string
	watchRecursive  bool
	watchQuality    int
	watchOnConflict string
	watchPreserveMD bool
	watchStripMD    bool
	watchRetry      int
	watchRetryDelay time.Duration
	watchInterval   time.Duration
	watchSettle     time.Duration
)

var watchCmd = &cobra.Command{
	Use:   "watch <dizin>",
	Short: "Klasörü izleyip yeni dosyaları otomatik dönüştür",
	Long: `Belirtilen klasörü polling yöntemiyle izler ve yeni/degisen dosyaları
otomatik dönüştürür.

Örnekler:
  fileconverter-cli watch ./incoming --from webp --to jpg
  fileconverter-cli watch ./videos --from mp4 --to gif --recursive --quality 80
  fileconverter-cli watch ./inbox --from png --to jpg --on-conflict versioned
  fileconverter-cli watch ./incoming --from mov --to mp4 --profile archive-lossless --preserve-metadata`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceDir := args[0]

		applyProfileDefault(cmd, "profile", &watchProfile)
		applyQualityDefault(cmd, "quality", &watchQuality)
		applyOnConflictDefault(cmd, "on-conflict", &watchOnConflict)
		applyMetadataDefault(cmd, "preserve-metadata", &watchPreserveMD, "strip-metadata", &watchStripMD)
		applyRetryDefaults(cmd, "retry", &watchRetry, "retry-delay", &watchRetryDelay)

		if p, ok, err := resolveProfile(watchProfile); err != nil {
			return err
		} else if ok {
			applyProfileToWatch(cmd, p)
			applyProfileMetadata(cmd, p, "preserve-metadata", &watchPreserveMD, "strip-metadata", &watchStripMD)
		}

		metadataMode, err := metadataModeFromFlags(watchPreserveMD, watchStripMD)
		if err != nil {
			return err
		}

		targetFormat := converter.NormalizeFormat(watchTo)
		if targetFormat == "" {
			return fmt.Errorf("hedef format belirtilmedi")
		}
		fromFormat := converter.NormalizeFormat(watchFrom)
		if fromFormat == "" {
			return fmt.Errorf("kaynak format belirtilmedi")
		}
		if _, err := converter.FindConverter(fromFormat, targetFormat); err != nil {
			return err
		}
		conflictPolicy := converter.NormalizeConflictPolicy(watchOnConflict)
		if conflictPolicy == "" {
			return fmt.Errorf("gecersiz on-conflict politikasi: %s", watchOnConflict)
		}

		w := convwatch.NewWatcher(sourceDir, fromFormat, watchRecursive, watchSettle)
		if err := w.Bootstrap(); err != nil {
			return err
		}

		pool := batch.NewPool(workers)
		pool.SetRetry(watchRetry, watchRetryDelay)

		ui.PrintInfo(fmt.Sprintf("İzleme başladı: %s (.%s -> .%s)", sourceDir, converter.FormatFilterLabel(fromFormat), targetFormat))
		ui.PrintInfo("Durdurmak için Ctrl+C kullanın.")

		ticker := time.NewTicker(watchInterval)
		defer ticker.Stop()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigCh)

		for {
			select {
			case <-ticker.C:
				files, err := w.Poll(time.Now())
				if err != nil {
					ui.PrintError(fmt.Sprintf("İzleme hatası: %s", err.Error()))
					continue
				}
				if len(files) == 0 {
					continue
				}

				jobs := make([]batch.Job, 0, len(files))
				reserved := make(map[string]struct{}, len(files))
				for _, f := range files {
					baseOutput := converter.BuildOutputPath(f, outputDir, targetFormat, "")
					resolvedOutput, skipReason, err := resolveBatchOutputPath(baseOutput, conflictPolicy, reserved)
					if err != nil {
						ui.PrintError(fmt.Sprintf("Çıktı yolu oluşturulamadı: %s", err.Error()))
						continue
					}
					jobs = append(jobs, batch.Job{
						InputPath:  f,
						OutputPath: resolvedOutput,
						From:       fromFormat,
						To:         targetFormat,
						SkipReason: skipReason,
						Options: converter.Options{
							Quality:      watchQuality,
							Verbose:      verbose,
							MetadataMode: metadataMode,
						},
					})
				}

				if len(jobs) == 0 {
					continue
				}

				startedAt := time.Now()
				results := pool.Execute(jobs)
				endedAt := time.Now()
				summary := batch.GetSummary(results, endedAt.Sub(startedAt))
				ui.PrintBatchSummary(summary.Total, summary.Succeeded, summary.Skipped, summary.Failed, summary.Duration)

				if len(summary.Errors) > 0 {
					ui.PrintError("Başarısız dönüşümler:")
					for _, e := range summary.Errors {
						fmt.Printf("  %s %s: %s (deneme: %d)\n", ui.IconError, e.InputFile, e.Error, e.Attempts)
					}
					fmt.Println()
				}

			case <-sigCh:
				ui.PrintInfo("İzleme durduruldu.")
				return nil
			}
		}
	},
}

func init() {
	watchCmd.Flags().StringVarP(&watchTo, "to", "t", "", "Hedef format (zorunlu)")
	watchCmd.Flags().StringVarP(&watchFrom, "from", "f", "", "Kaynak format (zorunlu)")
	watchCmd.Flags().StringVar(&watchProfile, "profile", "", "Hazır profil (ör: social-story, podcast-clean, archive-lossless)")
	watchCmd.Flags().BoolVarP(&watchRecursive, "recursive", "r", false, "Alt dizinleri de izle")
	watchCmd.Flags().IntVarP(&watchQuality, "quality", "q", 0, "Kalite seviyesi (1-100)")
	watchCmd.Flags().StringVar(&watchOnConflict, "on-conflict", converter.ConflictVersioned, "Çakışma politikası: overwrite, skip, versioned")
	watchCmd.Flags().BoolVar(&watchPreserveMD, "preserve-metadata", false, "Metadata bilgisini korumayı dene")
	watchCmd.Flags().BoolVar(&watchStripMD, "strip-metadata", false, "Metadata bilgisini temizle")
	watchCmd.Flags().IntVar(&watchRetry, "retry", 0, "Başarısız işler için otomatik tekrar sayısı")
	watchCmd.Flags().DurationVar(&watchRetryDelay, "retry-delay", 500*time.Millisecond, "Retry denemeleri arası bekleme (örn: 500ms, 2s)")
	watchCmd.Flags().DurationVar(&watchInterval, "interval", 2*time.Second, "Klasör tarama aralığı")
	watchCmd.Flags().DurationVar(&watchSettle, "settle", 1500*time.Millisecond, "Dosyanın stabil sayılması için bekleme süresi")

	watchCmd.MarkFlagRequired("to")
	watchCmd.MarkFlagRequired("from")

	rootCmd.AddCommand(watchCmd)
}
