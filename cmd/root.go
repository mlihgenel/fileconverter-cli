package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	// Converter modüllerini kaydet
	_ "github.com/mlihgenel/fileconverter-cli/internal/converter"
)

var (
	verbose   bool
	outputDir string
	workers   int

	appVersion = "1.1.0"
	appCommit  = "none"
	appDate    = "unknown"
)

// SetVersionInfo build-time version bilgisini ayarlar
func SetVersionInfo(version, commit, date string) {
	appVersion = version
	appCommit = commit
	appDate = date
}

var rootCmd = &cobra.Command{
	Use:   "fileconverter-cli",
	Short: "File Converter CLI - yerel dosya format donusturucu",
	Long: `File Converter CLI — Dosyalarınızı yerel ortamda güvenli bir şekilde dönüştürün.

Belge, ses, görsel ve video dosyalarını internet'e yüklemeden, tamamen yerel
olarak farklı formatlara dönüştürmenizi sağlar.

Interaktif ana menu:
  Dosya Donustur, Toplu Donustur, Boyutlandir, Toplu Boyutlandir

Desteklenen kategoriler:
  Belgeler:  MD, HTML, PDF, DOCX, TXT
  Ses:       MP3, WAV, OGG, FLAC, AAC, M4A, WMA  (FFmpeg gerekir)
  Gorseller: PNG, JPEG, WEBP, BMP, GIF, TIFF
  Videolar:  MP4, MOV, MKV, AVI, WEBM, M4V, WMV, FLV, GIF  (FFmpeg gerekir)

Örnekler:
  fileconverter-cli convert dosya.md --to pdf
  fileconverter-cli convert muzik.mp3 --to wav
  fileconverter-cli convert resim.png --to jpg --quality 90
  fileconverter-cli convert klip.mp4 --to mp4 --preset story --resize-mode pad
  fileconverter-cli convert klip.mp4 --to gif --quality 80
  fileconverter-cli batch ./belgeler --from md --to pdf
  fileconverter-cli resize-presets
  fileconverter-cli formats`,
	Version: appVersion,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Argümansız çalıştırıldığında interaktif mod başlat
		return RunInteractive()
	},
}

// Execute CLI'ı çalıştırır
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Detaylı çıktı modu")
	rootCmd.PersistentFlags().StringVarP(&outputDir, "output", "o", "", "Çıktı dizini (varsayılan: kaynak dizin)")
	rootCmd.PersistentFlags().IntVarP(&workers, "workers", "w", runtime.NumCPU(), "Paralel worker sayısı (batch modunda)")

	rootCmd.SetVersionTemplate(fmt.Sprintf(
		"FileConverter CLI v%s\nCommit: %s\nTarih:  %s\nGo:     %s\nOS:     %s/%s\n",
		appVersion, appCommit, appDate, runtime.Version(), runtime.GOOS, runtime.GOARCH,
	))

	// Hata mesajlarını özelleştir
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		fmt.Fprintf(os.Stderr, "Hata: %s\n\n", err.Error())
		cmd.Usage()
		return err
	})
}
