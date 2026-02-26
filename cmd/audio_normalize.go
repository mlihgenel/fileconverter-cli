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
	normalizeTargetLUFS float64
	normalizeTargetTP   float64
	normalizeTargetLRA  float64
	normalizeTo         string
	normalizeConflict   string
	normalizePreserveMD bool
	normalizeStripMD    bool
)

var audioCmd = &cobra.Command{
	Use:   "audio",
	Short: "Ses yardımcı komutları",
	Long:  `Ses dosyaları için yardımcı komutlar (normalize vb.).`,
}

var audioNormalizeCmd = &cobra.Command{
	Use:   "normalize <ses-dosyası>",
	Short: "Ses dosyasının ses seviyesini normalize eder",
	Long: `Ses dosyasının ses seviyesini EBU R128 standardına göre normalize eder.
FFmpeg loudnorm filtresi kullanarak hedef LUFS, True Peak ve LRA değerlerine
göre ses seviyesini ayarlar.

Örnekler:
  fileconverter-cli audio normalize podcast.mp3
  fileconverter-cli audio normalize song.wav --to mp3
  fileconverter-cli audio normalize voice.ogg --target-lufs -16
  fileconverter-cli audio normalize music.flac --target-lufs -14 --target-tp -1 --target-lra 9`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]
		if _, err := os.Stat(input); os.IsNotExist(err) {
			return fmt.Errorf("dosya bulunamadi: %s", input)
		}
		if !converter.IsFFmpegAvailable() {
			return fmt.Errorf("ses normalize için ffmpeg gerekli")
		}

		applyOnConflictDefault(cmd, "on-conflict", &normalizeConflict)
		applyMetadataDefault(cmd, "preserve-metadata", &normalizePreserveMD, "strip-metadata", &normalizeStripMD)

		metadataMode, err := metadataModeFromFlags(normalizePreserveMD, normalizeStripMD)
		if err != nil {
			return err
		}

		// LUFS/TP/LRA varsayılanları
		lufs := normalizeTargetLUFS
		if lufs == 0 {
			lufs = -14
		}
		tp := normalizeTargetTP
		if tp == 0 {
			tp = -1.5
		}
		lra := normalizeTargetLRA
		if lra == 0 {
			lra = 11
		}

		targetFormat := strings.ToLower(strings.TrimSpace(normalizeTo))
		if targetFormat == "" {
			targetFormat = converter.DetectFormat(input)
		}
		if targetFormat == "" {
			return fmt.Errorf("hedef format belirlenemedi")
		}

		outputPath := buildNormalizeOutputPath(input, targetFormat)
		conflict := converter.NormalizeConflictPolicy(normalizeConflict)
		if conflict == "" {
			return fmt.Errorf("gecersiz on-conflict politikasi: %s", normalizeConflict)
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
		ui.PrintInfo(fmt.Sprintf("Hedef: LUFS=%.1f, TP=%.1f, LRA=%.1f", lufs, tp, lra))
		started := time.Now()

		if err := runAudioNormalizeFFmpeg(input, outputPath, targetFormat, lufs, tp, lra, metadataMode, verbose); err != nil {
			ui.PrintError(err.Error())
			return err
		}

		ui.PrintSuccess("Ses normalize tamamlandı!")
		ui.PrintDuration(time.Since(started))
		return nil
	},
}

func init() {
	audioNormalizeCmd.Flags().Float64Var(&normalizeTargetLUFS, "target-lufs", -14, "Hedef loudness (LUFS, varsayılan: -14)")
	audioNormalizeCmd.Flags().Float64Var(&normalizeTargetTP, "target-tp", -1.5, "True peak limit (dB, varsayılan: -1.5)")
	audioNormalizeCmd.Flags().Float64Var(&normalizeTargetLRA, "target-lra", 11, "Loudness range (varsayılan: 11)")
	audioNormalizeCmd.Flags().StringVarP(&normalizeTo, "to", "t", "", "Çıktı ses formatı (varsayılan: kaynak format)")
	audioNormalizeCmd.Flags().StringVar(&normalizeConflict, "on-conflict", converter.ConflictVersioned, "Çakışma politikası: overwrite, skip, versioned")
	audioNormalizeCmd.Flags().BoolVar(&normalizePreserveMD, "preserve-metadata", false, "Metadata bilgisini korumayı dene")
	audioNormalizeCmd.Flags().BoolVar(&normalizeStripMD, "strip-metadata", false, "Metadata bilgisini temizle")

	audioCmd.AddCommand(audioNormalizeCmd)
	rootCmd.AddCommand(audioCmd)
}

func buildNormalizeOutputPath(input string, targetFormat string) string {
	base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	base = base + "_normalized"
	if strings.TrimSpace(outputDir) != "" {
		return filepath.Join(outputDir, base+"."+targetFormat)
	}
	return filepath.Join(filepath.Dir(input), base+"."+targetFormat)
}

func normalizeAudioCodecArgs(targetFormat string) []string {
	switch converter.NormalizeFormat(targetFormat) {
	case "mp3":
		return []string{"-c:a", "libmp3lame", "-b:a", "192k"}
	case "wav":
		return []string{"-c:a", "pcm_s16le"}
	case "ogg":
		return []string{"-c:a", "libvorbis", "-b:a", "192k"}
	case "flac":
		return []string{"-c:a", "flac"}
	case "aac", "m4a":
		return []string{"-c:a", "aac", "-b:a", "192k"}
	case "wma":
		return []string{"-c:a", "wmav2", "-b:a", "192k"}
	case "opus", "webm":
		return []string{"-c:a", "libopus", "-b:a", "192k"}
	default:
		return []string{"-b:a", "192k"}
	}
}

func runAudioNormalizeFFmpeg(input string, output string, targetFormat string, lufs float64, tp float64, lra float64, metadataMode string, verbose bool) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg bulunamadi")
	}

	args := []string{}
	if !verbose {
		args = append(args, "-loglevel", "error")
	}
	args = append(args, "-i", input, "-y")

	filter := fmt.Sprintf("loudnorm=I=%.1f:TP=%.1f:LRA=%.1f", lufs, tp, lra)
	args = append(args, "-af", filter)
	args = append(args, normalizeAudioCodecArgs(targetFormat)...)
	args = append(args, converter.MetadataFFmpegArgs(metadataMode)...)
	args = append(args, output)

	return runFFmpegCommand(ffmpegPath, args, "ses normalize ffmpeg hatasi")
}
