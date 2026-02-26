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
	mergeToFormat   string
	mergeQuality    int
	mergeName       string
	mergeConflict   string
	mergeReencode   bool
	mergePreserveMD bool
	mergeStripMD    bool
)

var mergeCmd = &cobra.Command{
	Use:   "merge <video1> <video2> [video3...]",
	Short: "Birden fazla videoyu sıralı olarak birleştirir",
	Long: `Birden fazla video dosyasını sıralı olarak tek bir dosyada birleştirir.

Aynı codec'teki videolar hızlı concat demuxer ile birleştirilir.
Farklı codec'lerde otomatik re-encode yapılır.

Örnekler:
  fileconverter-cli video merge part1.mp4 part2.mp4
  fileconverter-cli video merge part1.mp4 part2.mp4 part3.mp4 --name full_video
  fileconverter-cli video merge clip1.mov clip2.avi --to mp4
  fileconverter-cli video merge part1.mp4 part2.mp4 --reencode --quality 80`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, input := range args {
			if _, err := os.Stat(input); os.IsNotExist(err) {
				return fmt.Errorf("dosya bulunamadi: %s", input)
			}
		}
		if !converter.IsFFmpegAvailable() {
			return fmt.Errorf("video birleştirme için ffmpeg gerekli")
		}

		applyQualityDefault(cmd, "quality", &mergeQuality)
		applyOnConflictDefault(cmd, "on-conflict", &mergeConflict)
		applyMetadataDefault(cmd, "preserve-metadata", &mergePreserveMD, "strip-metadata", &mergeStripMD)

		metadataMode, err := metadataModeFromFlags(mergePreserveMD, mergeStripMD)
		if err != nil {
			return err
		}

		targetFormat := strings.ToLower(strings.TrimSpace(mergeToFormat))
		if targetFormat == "" {
			targetFormat = converter.DetectFormat(args[0])
		}
		if targetFormat == "" {
			targetFormat = "mp4"
		}

		// Codec tutarlılığını kontrol et
		canConcatDemux := !mergeReencode && checkCodecConsistency(args)

		outputPath := buildMergeOutputPath(args[0], targetFormat, mergeName)
		conflict := converter.NormalizeConflictPolicy(mergeConflict)
		if conflict == "" {
			return fmt.Errorf("gecersiz on-conflict politikasi: %s", mergeConflict)
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

		ui.PrintInfo(fmt.Sprintf("Birleştirilecek dosya sayısı: %d", len(args)))
		for i, f := range args {
			ui.PrintInfo(fmt.Sprintf("  [%d] %s", i+1, f))
		}
		ui.PrintInfo(fmt.Sprintf("Çıktı: %s", outputPath))

		if canConcatDemux {
			ui.PrintInfo("Mod: Concat demuxer (hızlı, codec copy)")
		} else {
			ui.PrintInfo("Mod: Re-encode (farklı codec'ler veya --reencode)")
		}

		started := time.Now()

		if canConcatDemux {
			err = runMergeConcatDemuxer(args, outputPath, metadataMode, verbose)
		} else {
			err = runMergeReencode(args, outputPath, targetFormat, mergeQuality, metadataMode, verbose)
		}
		if err != nil {
			ui.PrintError(err.Error())
			return err
		}

		ui.PrintSuccess("Video birleştirme tamamlandı!")
		ui.PrintDuration(time.Since(started))
		return nil
	},
}

func init() {
	mergeCmd.Flags().StringVarP(&mergeToFormat, "to", "t", "", "Çıktı video formatı (varsayılan: ilk dosyanın formatı)")
	mergeCmd.Flags().IntVarP(&mergeQuality, "quality", "q", 0, "Re-encode kalitesi (1-100)")
	mergeCmd.Flags().StringVarP(&mergeName, "name", "n", "", "Çıktı dosya adı (uzantısız)")
	mergeCmd.Flags().StringVar(&mergeConflict, "on-conflict", converter.ConflictVersioned, "Çakışma politikası: overwrite, skip, versioned")
	mergeCmd.Flags().BoolVar(&mergeReencode, "reencode", false, "Re-encode modunu zorla")
	mergeCmd.Flags().BoolVar(&mergePreserveMD, "preserve-metadata", false, "Metadata bilgisini korumayı dene")
	mergeCmd.Flags().BoolVar(&mergeStripMD, "strip-metadata", false, "Metadata bilgisini temizle")

	videoCmd.AddCommand(mergeCmd)
}

func buildMergeOutputPath(firstInput string, targetFormat string, customName string) string {
	base := strings.TrimSuffix(filepath.Base(firstInput), filepath.Ext(firstInput))
	if strings.TrimSpace(customName) != "" {
		base = customName
	} else {
		base = base + "_merged"
	}
	if strings.TrimSpace(outputDir) != "" {
		return filepath.Join(outputDir, base+"."+targetFormat)
	}
	return filepath.Join(filepath.Dir(firstInput), base+"."+targetFormat)
}

// checkCodecConsistency tüm video dosyalarının aynı codec'e sahip olup olmadığını kontrol eder.
func checkCodecConsistency(files []string) bool {
	if len(files) <= 1 {
		return true
	}

	firstCodec := probeVideoCodec(files[0])
	if firstCodec == "" {
		return false // codec algılanamıyorsa re-encode'a düş
	}

	for _, f := range files[1:] {
		if probeVideoCodec(f) != firstCodec {
			return false
		}
	}
	return true
}

// probeVideoCodec FFprobe ile video codec adını döner.
func probeVideoCodec(input string) string {
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		return ""
	}
	out, err := exec.Command(
		ffprobePath,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name",
		"-of", "default=noprint_wrappers=1:nokey=1",
		input,
	).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// writeConcatList concat demuxer için geçici dosya listesi oluşturur.
func writeConcatList(files []string, tempDir string) (string, error) {
	listPath := filepath.Join(tempDir, "concat_list.txt")
	var sb strings.Builder
	for _, f := range files {
		absPath, err := filepath.Abs(f)
		if err != nil {
			return "", err
		}
		// FFmpeg concat demuxer format: file 'path'
		escaped := strings.ReplaceAll(absPath, "'", "'\\''")
		sb.WriteString(fmt.Sprintf("file '%s'\n", escaped))
	}
	if err := os.WriteFile(listPath, []byte(sb.String()), 0644); err != nil {
		return "", err
	}
	return listPath, nil
}

func runMergeConcatDemuxer(inputs []string, output string, metadataMode string, verbose bool) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg bulunamadi")
	}

	tempDir, err := os.MkdirTemp("", "fileconverter-merge-*")
	if err != nil {
		return fmt.Errorf("geçici klasör oluşturulamadı: %w", err)
	}
	defer os.RemoveAll(tempDir)

	listPath, err := writeConcatList(inputs, tempDir)
	if err != nil {
		return fmt.Errorf("concat listesi oluşturulamadı: %w", err)
	}

	args := []string{}
	if !verbose {
		args = append(args, "-loglevel", "error")
	}
	args = append(args, "-f", "concat", "-safe", "0", "-i", listPath)
	args = append(args, "-c", "copy")
	args = append(args, converter.MetadataFFmpegArgs(metadataMode)...)
	args = append(args, "-y", output)

	return runFFmpegCommand(ffmpegPath, args, "video birleştirme (concat) ffmpeg hatasi")
}

func runMergeReencode(inputs []string, output string, targetFormat string, quality int, metadataMode string, verbose bool) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg bulunamadi")
	}

	tempDir, err := os.MkdirTemp("", "fileconverter-merge-*")
	if err != nil {
		return fmt.Errorf("geçici klasör oluşturulamadı: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Her dosyayı aynı formata dönüştürüp concat yap
	convertedParts := make([]string, 0, len(inputs))
	for i, input := range inputs {
		partPath := filepath.Join(tempDir, fmt.Sprintf("part_%02d.%s", i+1, targetFormat))
		partArgs := []string{}
		if !verbose {
			partArgs = append(partArgs, "-loglevel", "error")
		}
		partArgs = append(partArgs, "-i", input)
		partArgs = append(partArgs, mergeReencodeCodecArgs(targetFormat, quality)...)
		partArgs = append(partArgs, "-y", partPath)

		if err := runFFmpegCommand(ffmpegPath, partArgs, "video birleştirme ara dönüşüm hatasi"); err != nil {
			return err
		}
		convertedParts = append(convertedParts, partPath)
	}

	// Dönüştürülen parçaları concat demuxer ile birleştir
	listPath, err := writeConcatList(convertedParts, tempDir)
	if err != nil {
		return fmt.Errorf("concat listesi oluşturulamadı: %w", err)
	}

	args := []string{}
	if !verbose {
		args = append(args, "-loglevel", "error")
	}
	args = append(args, "-f", "concat", "-safe", "0", "-i", listPath)
	args = append(args, "-c", "copy")
	args = append(args, converter.MetadataFFmpegArgs(metadataMode)...)
	args = append(args, "-y", output)

	return runFFmpegCommand(ffmpegPath, args, "video birleştirme (final concat) ffmpeg hatasi")
}

func mergeCRF(quality int) int {
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

func mergeQScale(quality int) int {
	if quality <= 0 {
		return 5
	}
	switch {
	case quality <= 25:
		return 8
	case quality <= 50:
		return 6
	case quality <= 75:
		return 4
	default:
		return 2
	}
}

func mergeReencodeCodecArgs(targetFormat string, quality int) []string {
	crf := mergeCRF(quality)
	switch targetFormat {
	case "webm":
		webmCRF := crf + 6
		if webmCRF > 40 {
			webmCRF = 40
		}
		return []string{
			"-c:v", "libvpx-vp9", "-crf", fmt.Sprintf("%d", webmCRF), "-b:v", "0",
			"-c:a", "libopus", "-b:a", "128k",
		}
	case "avi":
		return []string{
			"-c:v", "mpeg4", "-q:v", fmt.Sprintf("%d", mergeQScale(quality)),
			"-c:a", "mp3", "-b:a", "192k",
		}
	default: // mp4, mov, mkv, m4v, flv
		return []string{
			"-c:v", "libx264", "-crf", fmt.Sprintf("%d", crf), "-preset", "medium", "-pix_fmt", "yuv420p",
			"-c:a", "aac", "-b:a", "128k",
		}
	}
}
