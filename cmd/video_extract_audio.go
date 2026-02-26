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
	extractAudioTo         string
	extractAudioQuality    int
	extractAudioCopy       bool
	extractAudioName       string
	extractAudioConflict   string
	extractAudioPreserveMD bool
	extractAudioStripMD    bool
)

// Desteklenen ses çıktı formatları
var extractAudioFormats = []string{"mp3", "wav", "ogg", "flac", "aac", "m4a", "opus"}

var extractAudioCmd = &cobra.Command{
	Use:   "extract-audio <video-dosyası>",
	Short: "Videodan ses kanalını ayrı dosya olarak çıkarır",
	Long: `Video dosyasından ses kanalını ayrı bir ses dosyası olarak çıkarır.

Örnekler:
  fileconverter-cli video extract-audio video.mp4
  fileconverter-cli video extract-audio video.mp4 --to wav
  fileconverter-cli video extract-audio video.mp4 --copy
  fileconverter-cli video extract-audio video.mp4 --to flac --name soundtrack
  fileconverter-cli video extract-audio video.mp4 --quality 90 --on-conflict overwrite`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]
		if _, err := os.Stat(input); os.IsNotExist(err) {
			return fmt.Errorf("dosya bulunamadi: %s", input)
		}
		if !converter.IsFFmpegAvailable() {
			return fmt.Errorf("ses çıkarma için ffmpeg gerekli")
		}

		applyQualityDefault(cmd, "quality", &extractAudioQuality)
		applyOnConflictDefault(cmd, "on-conflict", &extractAudioConflict)
		applyMetadataDefault(cmd, "preserve-metadata", &extractAudioPreserveMD, "strip-metadata", &extractAudioStripMD)

		metadataMode, err := metadataModeFromFlags(extractAudioPreserveMD, extractAudioStripMD)
		if err != nil {
			return err
		}

		targetFormat := strings.ToLower(strings.TrimSpace(extractAudioTo))
		if targetFormat == "" {
			targetFormat = "mp3"
		}
		if !isValidExtractAudioFormat(targetFormat) {
			return fmt.Errorf("desteklenmeyen ses formatı: %s (desteklenen: %s)", targetFormat, strings.Join(extractAudioFormats, ", "))
		}

		if extractAudioCopy && targetFormat != detectAudioStreamFormat(input) {
			ui.PrintWarning("--copy modu seçili fakat hedef format kaynak ses codec'i ile uyumsuz olabilir, sonuç hatalı olabilir.")
		}

		outputPath := buildExtractAudioOutputPath(input, targetFormat, extractAudioName)
		conflict := converter.NormalizeConflictPolicy(extractAudioConflict)
		if conflict == "" {
			return fmt.Errorf("gecersiz on-conflict politikasi: %s", extractAudioConflict)
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

		if err := runExtractAudioFFmpeg(input, outputPath, targetFormat, extractAudioQuality, extractAudioCopy, metadataMode, verbose); err != nil {
			ui.PrintError(err.Error())
			return err
		}

		ui.PrintSuccess("Ses çıkarma tamamlandı!")
		ui.PrintDuration(time.Since(started))
		return nil
	},
}

func init() {
	extractAudioCmd.Flags().StringVarP(&extractAudioTo, "to", "t", "mp3", "Hedef ses formatı (mp3, wav, ogg, flac, aac, m4a, opus)")
	extractAudioCmd.Flags().IntVarP(&extractAudioQuality, "quality", "q", 0, "Ses kalitesi (1-100)")
	extractAudioCmd.Flags().BoolVar(&extractAudioCopy, "copy", false, "Codec copy modu (re-encode yapmadan)")
	extractAudioCmd.Flags().StringVarP(&extractAudioName, "name", "n", "", "Çıktı dosya adı (uzantısız)")
	extractAudioCmd.Flags().StringVar(&extractAudioConflict, "on-conflict", converter.ConflictVersioned, "Çakışma politikası: overwrite, skip, versioned")
	extractAudioCmd.Flags().BoolVar(&extractAudioPreserveMD, "preserve-metadata", false, "Metadata bilgisini korumayı dene")
	extractAudioCmd.Flags().BoolVar(&extractAudioStripMD, "strip-metadata", false, "Metadata bilgisini temizle")

	videoCmd.AddCommand(extractAudioCmd)
}

func isValidExtractAudioFormat(format string) bool {
	for _, f := range extractAudioFormats {
		if f == format {
			return true
		}
	}
	return false
}

func buildExtractAudioOutputPath(input string, targetFormat string, customName string) string {
	base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	if strings.TrimSpace(customName) != "" {
		base = customName
	}
	if strings.TrimSpace(outputDir) != "" {
		return filepath.Join(outputDir, base+"."+targetFormat)
	}
	return filepath.Join(filepath.Dir(input), base+"."+targetFormat)
}

func extractAudioCodecArgs(targetFormat string, quality int, copyMode bool) []string {
	if copyMode {
		return []string{"-c:a", "copy"}
	}

	bitrate := "192k"
	if quality > 0 {
		switch {
		case quality <= 25:
			bitrate = "96k"
		case quality <= 50:
			bitrate = "128k"
		case quality <= 75:
			bitrate = "192k"
		default:
			bitrate = "320k"
		}
	}

	switch targetFormat {
	case "mp3":
		return []string{"-c:a", "libmp3lame", "-b:a", bitrate}
	case "wav":
		return []string{"-c:a", "pcm_s16le"}
	case "ogg":
		return []string{"-c:a", "libvorbis", "-b:a", bitrate}
	case "flac":
		return []string{"-c:a", "flac"}
	case "aac":
		return []string{"-c:a", "aac", "-b:a", bitrate}
	case "m4a":
		return []string{"-c:a", "aac", "-b:a", bitrate}
	case "opus":
		return []string{"-c:a", "libopus", "-b:a", bitrate}
	default:
		return []string{"-b:a", bitrate}
	}
}

func runExtractAudioFFmpeg(input string, output string, targetFormat string, quality int, copyMode bool, metadataMode string, verbose bool) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg bulunamadi")
	}

	args := []string{}
	if !verbose {
		args = append(args, "-loglevel", "error")
	}
	args = append(args, "-i", input)
	args = append(args, "-vn") // Video stream'i atla
	args = append(args, extractAudioCodecArgs(targetFormat, quality, copyMode)...)
	args = append(args, converter.MetadataFFmpegArgs(metadataMode)...)
	args = append(args, "-y", output)

	return runFFmpegCommand(ffmpegPath, args, "ses çıkarma ffmpeg hatasi")
}

// detectAudioStreamFormat FFprobe ile video dosyasındaki ses codec'ini algılar.
func detectAudioStreamFormat(input string) string {
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		return ""
	}
	out, err := exec.Command(
		ffprobePath,
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=codec_name",
		"-of", "default=noprint_wrappers=1:nokey=1",
		input,
	).Output()
	if err != nil {
		return ""
	}
	codec := strings.TrimSpace(string(out))
	// FFmpeg codec adını format adına çevir
	switch codec {
	case "aac":
		return "aac"
	case "mp3", "libmp3lame":
		return "mp3"
	case "vorbis":
		return "ogg"
	case "flac":
		return "flac"
	case "opus":
		return "opus"
	case "pcm_s16le", "pcm_s24le", "pcm_f32le":
		return "wav"
	default:
		return codec
	}
}
