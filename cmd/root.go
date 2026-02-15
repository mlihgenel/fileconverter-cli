package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	// Converter modüllerini kaydet
	_ "github.com/melihgenel/fileconverter/internal/converter"
)

var (
	verbose   bool
	outputDir string
	workers   int

	appVersion = "dev"
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
	Use:   "fileconverter",
	Short: "Yerel dosya format dönüştürücü",
	Long: `FileConverter — Dosyalarınızı yerel ortamda güvenli bir şekilde dönüştürün.

Belge, ses ve görsel dosyalarını internet'e yüklemeden, tamamen yerel
olarak farklı formatlara dönüştürmenizi sağlar.

Desteklenen kategoriler:
  📄 Belgeler:  MD, HTML, PDF, DOCX, TXT
  🎵 Ses:       MP3, WAV, OGG, FLAC, AAC, M4A, WMA  (FFmpeg gerektirir)
  🖼️  Görseller: PNG, JPEG, WEBP, BMP, GIF, TIFF

Örnekler:
  fileconverter convert dosya.md --to pdf
  fileconverter convert muzik.mp3 --to wav
  fileconverter convert resim.png --to jpg --quality 90
  fileconverter batch ./belgeler --from md --to pdf
  fileconverter formats`,
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
		fmt.Fprintf(os.Stderr, "❌ Hata: %s\n\n", err.Error())
		cmd.Usage()
		return err
	})
}
