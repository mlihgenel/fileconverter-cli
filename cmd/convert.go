package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlihgenel/fileconverter-cli/internal/converter"
	"github.com/mlihgenel/fileconverter-cli/internal/ui"
)

var (
	toFormat          string
	quality           int
	customName        string
	convertPreset     string
	convertWidth      float64
	convertHeight     float64
	convertUnit       string
	convertResizeDPI  float64
	convertResizeMode string
)

var convertCmd = &cobra.Command{
	Use:   "convert <dosya>",
	Short: "Tek bir dosyayı dönüştür",
	Long: `Bir dosyayı belirtilen formata dönüştürür.

Örnekler:
  fileconverter-cli convert README.md --to pdf
  fileconverter-cli convert belge.md --to html
  fileconverter-cli convert muzik.mp3 --to wav --quality 80
  fileconverter-cli convert resim.png --to jpg --quality 90 --output ./cikti/
  fileconverter-cli convert video.mp4 --to gif --quality 80
  fileconverter-cli convert dosya.pdf --to txt --name cikti_adi
  fileconverter-cli convert foto.jpg --to png --preset square --resize-mode pad
  fileconverter-cli convert klip.mp4 --to mp4 --preset story --resize-mode pad
  fileconverter-cli convert foto.jpg --to webp --width 12 --height 18 --unit cm --dpi 300`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputFile := args[0]

		// Dosya varlık kontrolü
		if _, err := os.Stat(inputFile); os.IsNotExist(err) {
			ui.PrintError(fmt.Sprintf("Dosya bulunamadı: %s", inputFile))
			return fmt.Errorf("dosya bulunamadı: %s", inputFile)
		}

		// Kaynak format algıla
		fromFormat := converter.DetectFormat(inputFile)
		if fromFormat == "" {
			ui.PrintError("Dosya formatı algılanamadı. Lütfen uzantılı bir dosya belirtin.")
			return fmt.Errorf("format algılanamadı")
		}

		// Hedef format kontrolü
		targetFormat := converter.NormalizeFormat(toFormat)
		if targetFormat == "" {
			ui.PrintError("Hedef format belirtilmedi. --to <format> kullanın.")
			return fmt.Errorf("hedef format belirtilmedi")
		}

		resizeSpec, err := converter.BuildResizeSpec(
			convertPreset,
			convertWidth,
			convertHeight,
			convertUnit,
			convertResizeMode,
			convertResizeDPI,
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

		// Converter bul
		conv, err := converter.FindConverter(fromFormat, targetFormat)
		if err != nil {
			ui.PrintError(err.Error())
			ui.PrintInfo(fmt.Sprintf("Desteklenen dönüşümleri görmek için: fileconverter-cli formats --from %s", fromFormat))
			return err
		}

		// Çıktı yolunu oluştur
		outputFile := converter.BuildOutputPath(inputFile, outputDir, targetFormat, customName)

		// Dönüşüm bilgisi
		if verbose {
			ui.PrintInfo(fmt.Sprintf("Dönüştürücü: %s", conv.Name()))
			ui.PrintInfo(fmt.Sprintf("Kaynak: %s (%s)", inputFile, fromFormat))
			ui.PrintInfo(fmt.Sprintf("Hedef:  %s (%s)", outputFile, targetFormat))
			if resizeSpec != nil {
				source := "manuel"
				if resizeSpec.Preset != "" {
					source = "preset: " + resizeSpec.Preset
				}
				ui.PrintInfo(fmt.Sprintf("Boyutlandırma: %dx%d (%s, mod: %s)", resizeSpec.Width, resizeSpec.Height, source, resizeSpec.Mode))
			}
		}

		ui.PrintConversion(inputFile, outputFile)

		// Dönüşümü yap
		start := time.Now()
		opts := converter.Options{
			Quality: quality,
			Verbose: verbose,
			Name:    customName,
			Resize:  resizeSpec,
		}

		if err := conv.Convert(inputFile, outputFile, opts); err != nil {
			ui.PrintError(fmt.Sprintf("Dönüşüm başarısız: %s", err.Error()))
			return err
		}

		duration := time.Since(start)
		ui.PrintSuccess(fmt.Sprintf("Dönüşüm tamamlandı!"))
		ui.PrintDuration(duration)

		// Dosya boyutu bilgisi
		if info, err := os.Stat(outputFile); err == nil {
			size := formatFileSize(info.Size())
			if verbose {
				ui.PrintInfo(fmt.Sprintf("Çıktı boyutu: %s", size))
			}
		}

		return nil
	},
}

func init() {
	convertCmd.Flags().StringVarP(&toFormat, "to", "t", "", "Hedef format (zorunlu, ör: pdf, docx, mp3)")
	convertCmd.Flags().IntVarP(&quality, "quality", "q", 0, "Kalite seviyesi (1-100, görsel/ses dönüşümleri için)")
	convertCmd.Flags().StringVarP(&customName, "name", "n", "", "Çıktı dosya adı (uzantısız)")
	convertCmd.Flags().StringVar(&convertPreset, "preset", "", "Hazır boyut preset'i (ör: story, square, fullhd, 1080x1920)")
	convertCmd.Flags().Float64Var(&convertWidth, "width", 0, "Manuel hedef genişlik")
	convertCmd.Flags().Float64Var(&convertHeight, "height", 0, "Manuel hedef yükseklik")
	convertCmd.Flags().StringVar(&convertUnit, "unit", "px", "Manuel ölçü birimi: px veya cm")
	convertCmd.Flags().Float64Var(&convertResizeDPI, "dpi", 96, "Birim cm ise kullanılacak DPI değeri")
	convertCmd.Flags().StringVar(&convertResizeMode, "resize-mode", "pad", "Boyutlandırma modu: pad, fit, fill, stretch")

	convertCmd.MarkFlagRequired("to")

	rootCmd.AddCommand(convertCmd)
}

// formatFileSize dosya boyutunu okunabilir formata çevirir
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
