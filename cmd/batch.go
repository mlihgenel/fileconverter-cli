package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlihgenel/fileconverter-cli/internal/batch"
	"github.com/mlihgenel/fileconverter-cli/internal/converter"
	"github.com/mlihgenel/fileconverter-cli/internal/ui"
)

var (
	batchTo         string
	batchFrom       string
	batchRecursive  bool
	batchDryRun     bool
	batchQuality    int
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
  fileconverter-cli batch ./fotograflar --from jpg --to webp --width 10 --height 15 --unit cm --dpi 300`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]

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

		// Dry-run modu
		if batchDryRun {
			ui.PrintInfo("Ön izleme modu (--dry-run) — dönüşüm yapılmayacak:")
			fmt.Println()
			for _, f := range files {
				outputFile := converter.BuildOutputPath(f, outputDir, targetFormat, "")
				ui.PrintConversion(f, outputFile)
			}
			fmt.Println()
			ui.PrintInfo(fmt.Sprintf("Toplam %d dosya dönüştürülecek.", len(files)))
			ui.PrintInfo("Dönüşümü başlatmak için --dry-run flag'ini kaldırın.")
			return nil
		}

		// İşleri oluştur
		jobs := make([]batch.Job, len(files))
		for i, f := range files {
			jobs[i] = batch.Job{
				InputPath:  f,
				OutputPath: converter.BuildOutputPath(f, outputDir, targetFormat, ""),
				From:       fromFormat,
				To:         targetFormat,
				Options: converter.Options{
					Quality: batchQuality,
					Verbose: verbose,
					Resize:  resizeSpec,
				},
			}
		}

		// Worker pool oluştur
		pool := batch.NewPool(workers)

		// Progress bar
		pb := ui.NewProgressBar(len(jobs), "Dönüştürülüyor")
		pool.OnProgress = func(completed, total int) {
			pb.Update(completed)
		}

		// Çalıştır
		fmt.Println()
		start := time.Now()
		results := pool.Execute(jobs)
		totalDuration := time.Since(start)

		// Sonuçları özetle
		summary := batch.GetSummary(results, totalDuration)
		ui.PrintBatchSummary(summary.Total, summary.Succeeded, summary.Failed, totalDuration)

		// Hataları göster
		if len(summary.Errors) > 0 {
			ui.PrintError("Başarısız dönüşümler:")
			for _, e := range summary.Errors {
				fmt.Printf("  %s %s: %s\n", ui.IconError, e.InputFile, e.Error)
			}
			fmt.Println()
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
	batchCmd.Flags().BoolVarP(&batchRecursive, "recursive", "r", false, "Alt dizinleri de tara")
	batchCmd.Flags().BoolVar(&batchDryRun, "dry-run", false, "Ön izleme — dönüşüm yapmadan listele")
	batchCmd.Flags().IntVarP(&batchQuality, "quality", "q", 0, "Kalite seviyesi (1-100)")
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
