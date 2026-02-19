package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlihgenel/fileconverter-cli/internal/converter"
	"github.com/mlihgenel/fileconverter-cli/internal/ui"
)

var (
	videoTrimStart      string
	videoTrimEnd        string
	videoTrimDuration   string
	videoTrimCodec      string
	videoTrimOutputFile string
	videoTrimName       string
	videoTrimToFormat   string
	videoTrimProfile    string
	videoTrimQuality    int
	videoTrimConflict   string
	videoTrimPreserveMD bool
	videoTrimStripMD    bool
)

var videoCmd = &cobra.Command{
	Use:   "video",
	Short: "Video yardımcı komutları (klip çıkarma vb.)",
	Long: `Video dosyaları için yardımcı komutlar.

Not: "trim" komutu videonun seçilen bölümünü yeni bir çıktı dosyası olarak çıkarır.
Orijinal videodan aralık silme (remove interval + stitch) yapmaz.`,
}

var videoTrimCmd = &cobra.Command{
	Use:   "trim <video-dosyasi>",
	Short: "Videodan zaman aralığını yeni klip dosyası olarak çıkarır",
	Long: `FFmpeg kullanarak videonun belirtilen aralığını yeni bir çıktı klibi olarak üretir.
Orijinal dosya değiştirilmez.

Örnekler:
  fileconverter-cli video trim input.mp4 --start 00:00:05 --duration 00:00:10
  fileconverter-cli video trim input.mp4 --start 00:01:00 --end 00:01:30 --codec reencode
  fileconverter-cli video trim input.mov --duration 15 --to mp4 --on-conflict versioned`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]
		if _, err := os.Stat(input); os.IsNotExist(err) {
			return fmt.Errorf("dosya bulunamadi: %s", input)
		}
		if !converter.IsFFmpegAvailable() {
			return fmt.Errorf("video trim için ffmpeg gerekli")
		}

		applyProfileDefault(cmd, "profile", &videoTrimProfile)
		applyQualityDefault(cmd, "quality", &videoTrimQuality)
		applyOnConflictDefault(cmd, "on-conflict", &videoTrimConflict)
		applyMetadataDefault(cmd, "preserve-metadata", &videoTrimPreserveMD, "strip-metadata", &videoTrimStripMD)

		if p, ok, err := resolveProfile(videoTrimProfile); err != nil {
			return err
		} else if ok {
			if p.Quality != nil && !cmd.Flags().Changed("quality") {
				videoTrimQuality = *p.Quality
			}
			if p.OnConflict != "" && !cmd.Flags().Changed("on-conflict") {
				videoTrimConflict = p.OnConflict
			}
			applyProfileMetadata(cmd, p, "preserve-metadata", &videoTrimPreserveMD, "strip-metadata", &videoTrimStripMD)
		}

		metadataMode, err := metadataModeFromFlags(videoTrimPreserveMD, videoTrimStripMD)
		if err != nil {
			return err
		}

		if err := validateTrimInput(videoTrimEnd, videoTrimDuration, videoTrimCodec); err != nil {
			return err
		}

		targetFormat := strings.TrimSpace(videoTrimToFormat)
		if targetFormat == "" {
			targetFormat = converter.DetectFormat(input)
		} else {
			targetFormat = converter.NormalizeFormat(targetFormat)
		}
		if targetFormat == "" {
			return fmt.Errorf("hedef format belirlenemedi")
		}

		outputPath := buildTrimOutputPath(input, targetFormat, videoTrimName, videoTrimOutputFile)
		conflict := converter.NormalizeConflictPolicy(videoTrimConflict)
		if conflict == "" {
			return fmt.Errorf("gecersiz on-conflict politikasi: %s", videoTrimConflict)
		}
		outputPath, skip, err := converter.ResolveOutputPathConflict(outputPath, conflict)
		if err != nil {
			return err
		}
		if skip {
			ui.PrintWarning(fmt.Sprintf("Çıktı dosyası mevcut, atlandı: %s", outputPath))
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return err
		}

		ui.PrintConversion(input, outputPath)
		started := time.Now()
		if err := runTrimFFmpeg(input, outputPath, videoTrimStart, videoTrimEnd, videoTrimDuration, videoTrimCodec, videoTrimQuality, metadataMode, verbose); err != nil {
			ui.PrintError(err.Error())
			return err
		}
		ui.PrintSuccess("Video klip çıkarma tamamlandı!")
		ui.PrintDuration(time.Since(started))
		return nil
	},
}

func init() {
	videoTrimCmd.Flags().StringVar(&videoTrimStart, "start", "0", "Klibin başlangıç zamanı (örn: 00:01:05)")
	videoTrimCmd.Flags().StringVar(&videoTrimEnd, "end", "", "Bitiş zamanı (örn: 00:02:00)")
	videoTrimCmd.Flags().StringVar(&videoTrimDuration, "duration", "", "Klip süresi (örn: 15, 00:00:15)")
	videoTrimCmd.Flags().StringVar(&videoTrimCodec, "codec", "copy", "Klip çıkarma codec modu: copy veya reencode")
	videoTrimCmd.Flags().StringVar(&videoTrimOutputFile, "output-file", "", "Tam çıktı dosya yolu")
	videoTrimCmd.Flags().StringVarP(&videoTrimName, "name", "n", "", "Çıktı klip adı (uzantısız)")
	videoTrimCmd.Flags().StringVar(&videoTrimToFormat, "to", "", "Hedef format (örn: mp4, mov)")
	videoTrimCmd.Flags().StringVar(&videoTrimProfile, "profile", "", "Hazır profil (ör: social-story, podcast-clean, archive-lossless)")
	videoTrimCmd.Flags().IntVarP(&videoTrimQuality, "quality", "q", 0, "Reencode modunda kalite seviyesi (1-100)")
	videoTrimCmd.Flags().StringVar(&videoTrimConflict, "on-conflict", converter.ConflictVersioned, "Çakışma politikası: overwrite, skip, versioned")
	videoTrimCmd.Flags().BoolVar(&videoTrimPreserveMD, "preserve-metadata", false, "Metadata bilgisini korumayı dene")
	videoTrimCmd.Flags().BoolVar(&videoTrimStripMD, "strip-metadata", false, "Metadata bilgisini temizle")

	videoCmd.AddCommand(videoTrimCmd)
	rootCmd.AddCommand(videoCmd)
}

func validateTrimInput(end string, duration string, codec string) error {
	if strings.TrimSpace(end) != "" && strings.TrimSpace(duration) != "" {
		return fmt.Errorf("--end ve --duration birlikte kullanılamaz")
	}
	c := strings.ToLower(strings.TrimSpace(codec))
	if c != "copy" && c != "reencode" {
		return fmt.Errorf("gecersiz codec modu: %s (copy|reencode)", codec)
	}
	return nil
}

func buildTrimOutputPath(input string, targetFormat string, customName string, explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		return explicit
	}
	base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	if strings.TrimSpace(customName) != "" {
		base = customName
	} else {
		base += "_trim"
	}
	if strings.TrimSpace(outputDir) != "" {
		return filepath.Join(outputDir, base+"."+targetFormat)
	}
	return filepath.Join(filepath.Dir(input), base+"."+targetFormat)
}

func runTrimFFmpeg(input string, output string, start string, end string, duration string, codec string, quality int, metadataMode string, verbose bool) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg bulunamadi")
	}

	args := []string{}
	if !verbose {
		args = append(args, "-loglevel", "error")
	}
	args = append(args, "-i", input, "-y")

	if strings.TrimSpace(start) != "" && strings.TrimSpace(start) != "0" {
		args = append(args, "-ss", strings.TrimSpace(start))
	}
	if strings.TrimSpace(end) != "" {
		args = append(args, "-to", strings.TrimSpace(end))
	}
	if strings.TrimSpace(duration) != "" {
		args = append(args, "-t", strings.TrimSpace(duration))
	}

	switch strings.ToLower(strings.TrimSpace(codec)) {
	case "copy":
		args = append(args, "-c", "copy")
	case "reencode":
		crf := trimCRF(quality)
		args = append(args,
			"-c:v", "libx264",
			"-crf", fmt.Sprintf("%d", crf),
			"-preset", "medium",
			"-pix_fmt", "yuv420p",
			"-c:a", "aac",
			"-b:a", "128k",
		)
	}

	args = append(args, converter.MetadataFFmpegArgs(metadataMode)...)
	args = append(args, output)

	cmd := exec.Command(ffmpegPath, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("video trim ffmpeg hatasi: %s\n%s", err.Error(), string(out))
	}
	return nil
}

func trimCRF(quality int) int {
	if quality <= 0 {
		return 23
	}
	switch {
	case quality <= 25:
		return 30
	case quality <= 50:
		return 27
	case quality <= 75:
		return 24
	default:
		return 20
	}
}
