package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlihgenel/fileconverter-cli/internal/converter"
	"github.com/mlihgenel/fileconverter-cli/internal/ui"
)

var (
	snapshotAt       string
	snapshotTo       string
	snapshotQuality  int
	snapshotName     string
	snapshotConflict string
)

// Desteklenen snapshot çıktı formatları
var snapshotOutputFormats = []string{"png", "jpg", "webp", "bmp"}

var snapshotCmd = &cobra.Command{
	Use:   "snapshot <video-dosyası>",
	Short: "Videonun belirli bir anından kare yakalar",
	Long: `Video dosyasından belirli bir zaman noktasında tek kare çıkarır.

Zaman belirtme yöntemleri:
  - Saniye: --at 30 veya --at 5.5
  - Zaman formatı: --at 00:01:30
  - Yüzde: --at %50 (videonun ortasından)

Örnekler:
  fileconverter-cli video snapshot video.mp4 --at 10
  fileconverter-cli video snapshot video.mp4 --at 00:01:30 --to jpg
  fileconverter-cli video snapshot video.mp4 --at %50 --name thumbnail
  fileconverter-cli video snapshot video.mp4 --at 5.5 --to webp --quality 90`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]
		if _, err := os.Stat(input); os.IsNotExist(err) {
			return fmt.Errorf("dosya bulunamadi: %s", input)
		}
		if !converter.IsFFmpegAvailable() {
			return fmt.Errorf("snapshot için ffmpeg gerekli")
		}

		applyQualityDefault(cmd, "quality", &snapshotQuality)
		applyOnConflictDefault(cmd, "on-conflict", &snapshotConflict)

		targetFormat := strings.ToLower(strings.TrimSpace(snapshotTo))
		if targetFormat == "" {
			targetFormat = "png"
		}
		if !isValidSnapshotFormat(targetFormat) {
			return fmt.Errorf("desteklenmeyen görsel formatı: %s (desteklenen: %s)", targetFormat, strings.Join(snapshotOutputFormats, ", "))
		}

		atValue := strings.TrimSpace(snapshotAt)
		if atValue == "" {
			return fmt.Errorf("--at flag'i zorunludur (örn: --at 10, --at 00:01:30, --at %%50)")
		}

		seekSeconds, err := resolveSnapshotTime(atValue, input)
		if err != nil {
			return err
		}

		outputPath := buildSnapshotOutputPath(input, targetFormat, snapshotName, seekSeconds)
		conflict := converter.NormalizeConflictPolicy(snapshotConflict)
		if conflict == "" {
			return fmt.Errorf("gecersiz on-conflict politikasi: %s", snapshotConflict)
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
		ui.PrintInfo(fmt.Sprintf("Zaman noktası: %s", formatTrimSecondsHuman(seekSeconds)))
		started := time.Now()

		if err := runSnapshotFFmpeg(input, outputPath, seekSeconds, targetFormat, snapshotQuality, verbose); err != nil {
			ui.PrintError(err.Error())
			return err
		}

		ui.PrintSuccess("Kare yakalama tamamlandı!")
		ui.PrintDuration(time.Since(started))
		return nil
	},
}

func init() {
	snapshotCmd.Flags().StringVar(&snapshotAt, "at", "", "Zaman noktası (saniye, HH:MM:SS veya %yüzde)")
	snapshotCmd.Flags().StringVarP(&snapshotTo, "to", "t", "png", "Çıktı görsel formatı (png, jpg, webp, bmp)")
	snapshotCmd.Flags().IntVarP(&snapshotQuality, "quality", "q", 0, "Görsel kalitesi (1-100)")
	snapshotCmd.Flags().StringVarP(&snapshotName, "name", "n", "", "Çıktı dosya adı (uzantısız)")
	snapshotCmd.Flags().StringVar(&snapshotConflict, "on-conflict", converter.ConflictVersioned, "Çakışma politikası: overwrite, skip, versioned")

	videoCmd.AddCommand(snapshotCmd)
}

func isValidSnapshotFormat(format string) bool {
	for _, f := range snapshotOutputFormats {
		if f == format {
			return true
		}
	}
	return false
}

func buildSnapshotOutputPath(input string, targetFormat string, customName string, seekSeconds float64) string {
	base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	if strings.TrimSpace(customName) != "" {
		base = customName
	} else {
		// Zaman bilgisini dosya adına ekle
		timeStr := fmt.Sprintf("%.0f", seekSeconds)
		base = base + "_snapshot_" + timeStr + "s"
	}
	if strings.TrimSpace(outputDir) != "" {
		return filepath.Join(outputDir, base+"."+targetFormat)
	}
	return filepath.Join(filepath.Dir(input), base+"."+targetFormat)
}

// resolveSnapshotTime --at değerini saniye cinsine çevirir.
// Desteklenen formatlar: "30", "5.5", "00:01:30", "%50"
func resolveSnapshotTime(at string, input string) (float64, error) {
	at = strings.TrimSpace(at)
	if at == "" {
		return 0, fmt.Errorf("zaman değeri boş olamaz")
	}

	// Yüzde modu: %50
	if strings.HasPrefix(at, "%") {
		percentStr := strings.TrimPrefix(at, "%")
		percent, err := strconv.ParseFloat(percentStr, 64)
		if err != nil || percent < 0 || percent > 100 {
			return 0, fmt.Errorf("geçersiz yüzde değeri: %s (0-100 arası olmalı)", at)
		}

		duration, hasDuration := probeMediaDurationSeconds(input)
		if !hasDuration {
			return 0, fmt.Errorf("yüzde modu için video süresi alınamadı (ffprobe gerekli)")
		}
		return duration * percent / 100.0, nil
	}

	// Zaman formatı: HH:MM:SS veya saniye
	normalized, err := normalizeVideoTrimTime(at, true)
	if err != nil {
		return 0, fmt.Errorf("geçersiz zaman değeri: %s", at)
	}
	seconds, err := parseVideoTrimToSeconds(normalized)
	if err != nil {
		return 0, fmt.Errorf("geçersiz zaman değeri: %s", at)
	}
	if seconds < 0 {
		return 0, fmt.Errorf("zaman değeri negatif olamaz")
	}
	return seconds, nil
}

func snapshotCodecArgs(targetFormat string, quality int) []string {
	switch targetFormat {
	case "jpg":
		q := 2 // FFmpeg qscale: 2=yüksek kalite, 31=düşük
		if quality > 0 {
			switch {
			case quality <= 25:
				q = 15
			case quality <= 50:
				q = 8
			case quality <= 75:
				q = 4
			default:
				q = 2
			}
		}
		return []string{"-q:v", strconv.Itoa(q)}
	case "webp":
		qVal := 80
		if quality > 0 {
			qVal = quality
		}
		return []string{"-quality", strconv.Itoa(qVal)}
	default: // png, bmp
		return []string{}
	}
}

func runSnapshotFFmpeg(input string, output string, seekSeconds float64, targetFormat string, quality int, verbose bool) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg bulunamadi")
	}

	args := []string{}
	if !verbose {
		args = append(args, "-loglevel", "error")
	}

	// -ss before -i for fast seeking
	seekStr := formatSecondsForFFmpeg(seekSeconds)
	args = append(args, "-ss", seekStr)
	args = append(args, "-i", input)
	args = append(args, "-vframes", "1")
	args = append(args, snapshotCodecArgs(targetFormat, quality)...)
	args = append(args, "-y", output)

	return runFFmpegCommand(ffmpegPath, args, "snapshot ffmpeg hatasi")
}
