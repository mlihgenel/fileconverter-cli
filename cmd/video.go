package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
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
	videoTrimRanges     string
	videoTrimMode       string
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

const (
	trimModeClip   = "clip"
	trimModeRemove = "remove"
)

type trimRange struct {
	Start float64
	End   float64
}

var videoCmd = &cobra.Command{
	Use:   "video",
	Short: "Video yardımcı komutları (klip çıkarma ve aralık silme)",
	Long: `Video dosyaları için yardımcı komutlar.

Not: "trim" komutu iki mod destekler:
  - clip: seçilen aralığı yeni klip olarak çıkarır
  - remove: seçilen aralığı siler, kalan parçaları birleştirir`,
}

var videoTrimCmd = &cobra.Command{
	Use:   "trim <video-dosyasi>",
	Short: "Videoda aralık çıkarma veya aralık silme işlemi yapar",
	Long: `FFmpeg ile iki modda çalışır:
  - clip: belirtilen aralığı yeni bir klip olarak üretir (orijinali değiştirmez)
  - remove: belirtilen aralığı siler, kalan bölümleri birleştirip yeni dosya üretir

Örnekler:
  fileconverter-cli video trim input.mp4 --start 00:00:05 --duration 00:00:10
  fileconverter-cli video trim input.mp4 --mode remove --start 00:00:23 --duration 2
  fileconverter-cli video trim input.mp4 --mode remove --ranges "00:00:05-00:00:08,00:00:20-00:00:25"
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

		if err := validateTrimInput(videoTrimMode, videoTrimEnd, videoTrimDuration, videoTrimRanges, videoTrimCodec); err != nil {
			return err
		}
		mode := normalizeTrimMode(videoTrimMode)
		codec := normalizeTrimCodec(videoTrimCodec)
		if strings.TrimSpace(videoTrimRanges) != "" && (cmd.Flags().Changed("start") || cmd.Flags().Changed("end") || cmd.Flags().Changed("duration")) {
			return fmt.Errorf("--ranges kullanırken --start/--end/--duration birlikte kullanılamaz")
		}
		startValue := ""
		endValue := ""
		durationValue := ""
		removeRanges := []trimRange(nil)
		if strings.TrimSpace(videoTrimRanges) != "" {
			removeRanges, err = parseTrimRangesSpec(videoTrimRanges)
			if err != nil {
				return err
			}
		} else {
			startValue, endValue, durationValue, _, _, err = resolveTrimRange(videoTrimStart, videoTrimEnd, videoTrimDuration, mode)
			if err != nil {
				return err
			}
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

		outputPath := buildTrimOutputPath(input, targetFormat, videoTrimName, videoTrimOutputFile, mode)
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
		if mode == trimModeClip {
			err = runTrimFFmpeg(input, outputPath, startValue, endValue, durationValue, codec, videoTrimQuality, metadataMode, verbose)
		} else {
			if len(removeRanges) > 0 {
				err = runTrimRemoveRangesFFmpeg(input, outputPath, removeRanges, codec, videoTrimQuality, metadataMode, verbose)
			} else {
				err = runTrimRemoveFFmpeg(input, outputPath, startValue, endValue, durationValue, codec, videoTrimQuality, metadataMode, verbose)
			}
		}
		if err != nil {
			ui.PrintError(err.Error())
			return err
		}
		if mode == trimModeClip {
			ui.PrintSuccess("Video klip çıkarma tamamlandı!")
		} else {
			ui.PrintSuccess("Video aralığı silme + birleştirme tamamlandı!")
		}
		ui.PrintDuration(time.Since(started))
		return nil
	},
}

func init() {
	videoTrimCmd.Flags().StringVar(&videoTrimStart, "start", "0", "İşlem başlangıç zamanı (örn: 00:01:05)")
	videoTrimCmd.Flags().StringVar(&videoTrimEnd, "end", "", "Bitiş zamanı (örn: 00:02:00)")
	videoTrimCmd.Flags().StringVar(&videoTrimDuration, "duration", "", "İşlem süresi (örn: 15, 00:00:15)")
	videoTrimCmd.Flags().StringVar(&videoTrimRanges, "ranges", "", "Sadece remove modunda çoklu aralık listesi (örn: 00:00:05-00:00:08,00:00:20-00:00:25)")
	videoTrimCmd.Flags().StringVar(&videoTrimMode, "mode", trimModeClip, "İşlem modu: clip veya remove")
	videoTrimCmd.Flags().StringVar(&videoTrimCodec, "codec", "copy", "Codec modu: copy veya reencode")
	videoTrimCmd.Flags().StringVar(&videoTrimOutputFile, "output-file", "", "Tam çıktı dosya yolu")
	videoTrimCmd.Flags().StringVarP(&videoTrimName, "name", "n", "", "Çıktı dosya adı (uzantısız)")
	videoTrimCmd.Flags().StringVar(&videoTrimToFormat, "to", "", "Hedef format (örn: mp4, mov)")
	videoTrimCmd.Flags().StringVar(&videoTrimProfile, "profile", "", "Hazır profil (ör: social-story, podcast-clean, archive-lossless)")
	videoTrimCmd.Flags().IntVarP(&videoTrimQuality, "quality", "q", 0, "Reencode modunda kalite seviyesi (1-100)")
	videoTrimCmd.Flags().StringVar(&videoTrimConflict, "on-conflict", converter.ConflictVersioned, "Çakışma politikası: overwrite, skip, versioned")
	videoTrimCmd.Flags().BoolVar(&videoTrimPreserveMD, "preserve-metadata", false, "Metadata bilgisini korumayı dene")
	videoTrimCmd.Flags().BoolVar(&videoTrimStripMD, "strip-metadata", false, "Metadata bilgisini temizle")

	videoCmd.AddCommand(videoTrimCmd)
	rootCmd.AddCommand(videoCmd)
}

func validateTrimInput(mode string, end string, duration string, ranges string, codec string) error {
	if strings.TrimSpace(end) != "" && strings.TrimSpace(duration) != "" {
		return fmt.Errorf("--end ve --duration birlikte kullanılamaz")
	}
	normalizedMode := normalizeTrimMode(mode)
	if normalizedMode == "" {
		return fmt.Errorf("gecersiz mode: %s (clip|remove)", mode)
	}
	c := normalizeTrimCodec(codec)
	if c == "" {
		return fmt.Errorf("gecersiz codec modu: %s (copy|reencode)", codec)
	}
	hasRanges := strings.TrimSpace(ranges) != ""
	if hasRanges && normalizedMode != trimModeRemove {
		return fmt.Errorf("--ranges sadece remove modunda kullanılabilir")
	}
	if hasRanges && (strings.TrimSpace(end) != "" || strings.TrimSpace(duration) != "") {
		return fmt.Errorf("--ranges ile --end/--duration birlikte kullanılamaz")
	}
	if normalizedMode == trimModeRemove && !hasRanges && strings.TrimSpace(end) == "" && strings.TrimSpace(duration) == "" {
		return fmt.Errorf("remove modunda --end veya --duration zorunludur")
	}
	return nil
}

func normalizeTrimMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", trimModeClip:
		return trimModeClip
	case trimModeRemove:
		return trimModeRemove
	default:
		return ""
	}
}

func normalizeTrimCodec(codec string) string {
	c := strings.ToLower(strings.TrimSpace(codec))
	if c != "copy" && c != "reencode" {
		return ""
	}
	return c
}

func resolveTrimRange(start string, end string, duration string, mode string) (string, string, string, float64, float64, error) {
	startRaw := strings.TrimSpace(start)
	if startRaw == "" {
		startRaw = "0"
	}
	startValue, err := normalizeVideoTrimTime(startRaw, true)
	if err != nil {
		return "", "", "", 0, 0, fmt.Errorf("geçersiz --start değeri: %s", start)
	}
	startSec, err := parseVideoTrimToSeconds(startValue)
	if err != nil {
		return "", "", "", 0, 0, fmt.Errorf("geçersiz --start değeri: %s", start)
	}

	endValue := ""
	durationValue := ""
	endSec := 0.0

	if strings.TrimSpace(end) != "" {
		endValue, err = normalizeVideoTrimTime(end, true)
		if err != nil {
			return "", "", "", 0, 0, fmt.Errorf("geçersiz --end değeri: %s", end)
		}
		endSec, err = parseVideoTrimToSeconds(endValue)
		if err != nil {
			return "", "", "", 0, 0, fmt.Errorf("geçersiz --end değeri: %s", end)
		}
	} else if strings.TrimSpace(duration) != "" {
		durationValue, err = normalizeVideoTrimTime(duration, false)
		if err != nil {
			return "", "", "", 0, 0, fmt.Errorf("geçersiz --duration değeri: %s", duration)
		}
		durationSec, err := parseVideoTrimToSeconds(durationValue)
		if err != nil {
			return "", "", "", 0, 0, fmt.Errorf("geçersiz --duration değeri: %s", duration)
		}
		endSec = startSec + durationSec
	}

	if endSec > 0 && endSec <= startSec {
		return "", "", "", 0, 0, fmt.Errorf("bitiş zamanı başlangıçtan büyük olmalıdır")
	}
	if mode == trimModeRemove && endSec <= 0 {
		return "", "", "", 0, 0, fmt.Errorf("remove modunda geçerli bir bitiş zamanı gerekir")
	}

	return startValue, endValue, durationValue, startSec, endSec, nil
}

func parseTrimRangesSpec(spec string) ([]trimRange, error) {
	tokens := strings.Split(spec, ",")
	ranges := make([]trimRange, 0, len(tokens))

	for _, token := range tokens {
		raw := strings.TrimSpace(token)
		if raw == "" {
			continue
		}
		parts := strings.SplitN(raw, "-", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("geçersiz aralık: %s (örn: 00:00:05-00:00:08)", raw)
		}

		startValue, err := normalizeVideoTrimTime(strings.TrimSpace(parts[0]), true)
		if err != nil {
			return nil, fmt.Errorf("geçersiz aralık başlangıcı: %s", strings.TrimSpace(parts[0]))
		}
		endValue, err := normalizeVideoTrimTime(strings.TrimSpace(parts[1]), true)
		if err != nil {
			return nil, fmt.Errorf("geçersiz aralık bitişi: %s", strings.TrimSpace(parts[1]))
		}

		startSec, err := parseVideoTrimToSeconds(startValue)
		if err != nil {
			return nil, fmt.Errorf("geçersiz aralık başlangıcı: %s", strings.TrimSpace(parts[0]))
		}
		endSec, err := parseVideoTrimToSeconds(endValue)
		if err != nil {
			return nil, fmt.Errorf("geçersiz aralık bitişi: %s", strings.TrimSpace(parts[1]))
		}
		if endSec <= startSec {
			return nil, fmt.Errorf("aralıkta bitiş başlangıçtan büyük olmalı: %s", raw)
		}

		ranges = append(ranges, trimRange{Start: startSec, End: endSec})
	}

	if len(ranges) == 0 {
		return nil, fmt.Errorf("en az bir aralık belirtmelisiniz (--ranges)")
	}
	return mergeTrimRanges(ranges), nil
}

func mergeTrimRanges(ranges []trimRange) []trimRange {
	if len(ranges) == 0 {
		return nil
	}
	cloned := make([]trimRange, len(ranges))
	copy(cloned, ranges)

	sort.Slice(cloned, func(i, j int) bool {
		if cloned[i].Start == cloned[j].Start {
			return cloned[i].End < cloned[j].End
		}
		return cloned[i].Start < cloned[j].Start
	})

	const epsilon = 0.001
	merged := []trimRange{cloned[0]}
	for _, r := range cloned[1:] {
		last := &merged[len(merged)-1]
		if r.Start <= last.End+epsilon {
			if r.End > last.End {
				last.End = r.End
			}
			continue
		}
		merged = append(merged, r)
	}
	return merged
}

func buildTrimOutputPath(input string, targetFormat string, customName string, explicit string, mode string) string {
	if strings.TrimSpace(explicit) != "" {
		return explicit
	}
	base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	if strings.TrimSpace(customName) != "" {
		base = customName
	} else {
		if mode == trimModeRemove {
			base += "_cut"
		} else {
			base += "_trim"
		}
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

	startSec := 0.0
	if strings.TrimSpace(start) != "" {
		startSec, err = parseVideoTrimToSeconds(start)
		if err != nil {
			return fmt.Errorf("geçersiz başlangıç zamanı")
		}
	}

	endSec := 0.0
	hasRequestedEnd := false
	if strings.TrimSpace(end) != "" {
		endSec, err = parseVideoTrimToSeconds(end)
		if err != nil {
			return fmt.Errorf("geçersiz bitiş zamanı")
		}
		hasRequestedEnd = true
	} else if strings.TrimSpace(duration) != "" {
		durationSec, parseErr := parseVideoTrimToSeconds(duration)
		if parseErr != nil {
			return fmt.Errorf("geçersiz süre değeri")
		}
		endSec = startSec + durationSec
		hasRequestedEnd = true
	}

	startSec, endSec, err = adjustTrimWindowByDuration(input, startSec, endSec, trimModeClip)
	if err != nil {
		return err
	}
	start = formatSecondsForFFmpeg(startSec)
	if hasRequestedEnd {
		end = formatSecondsForFFmpeg(endSec)
		duration = ""
	}

	args := []string{}
	if !verbose {
		args = append(args, "-loglevel", "error")
	}
	args = append(args, "-i", input)

	if strings.TrimSpace(start) != "" && strings.TrimSpace(start) != "0" {
		args = append(args, "-ss", strings.TrimSpace(start))
	}
	if strings.TrimSpace(end) != "" {
		args = append(args, "-to", strings.TrimSpace(end))
	}
	if strings.TrimSpace(duration) != "" {
		args = append(args, "-t", strings.TrimSpace(duration))
	}

	args = append(args, trimCodecArgs(codec, quality)...)

	args = append(args, converter.MetadataFFmpegArgs(metadataMode)...)
	args = append(args, "-y")
	args = append(args, output)

	if err := runFFmpegCommand(ffmpegPath, args, "video trim ffmpeg hatasi"); err != nil {
		return err
	}
	return nil
}

func runTrimRemoveFFmpeg(input string, output string, start string, end string, duration string, codec string, quality int, metadataMode string, verbose bool) error {
	startSec, err := parseVideoTrimToSeconds(start)
	if err != nil {
		return fmt.Errorf("geçersiz başlangıç zamanı")
	}

	endSec := 0.0
	if strings.TrimSpace(end) != "" {
		endSec, err = parseVideoTrimToSeconds(end)
	} else {
		durationSec, parseErr := parseVideoTrimToSeconds(duration)
		if parseErr != nil {
			return fmt.Errorf("geçersiz süre değeri")
		}
		endSec = startSec + durationSec
	}
	if err != nil {
		return fmt.Errorf("geçersiz bitiş zamanı")
	}
	if endSec <= startSec {
		return fmt.Errorf("bitiş zamanı başlangıçtan büyük olmalıdır")
	}

	return runTrimRemoveRangesFFmpeg(input, output, []trimRange{{Start: startSec, End: endSec}}, codec, quality, metadataMode, verbose)
}

type keepSegment struct {
	Start  float64
	End    float64
	HasEnd bool
}

func runTrimRemoveRangesFFmpeg(input string, output string, ranges []trimRange, codec string, quality int, metadataMode string, verbose bool) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg bulunamadi")
	}
	if len(ranges) == 0 {
		return fmt.Errorf("remove işlemi için en az bir aralık gerekir")
	}

	ranges = mergeTrimRanges(ranges)
	durationSec, hasDuration := probeMediaDurationSeconds(input)
	if hasDuration {
		ranges, err = clampTrimRangesToDuration(ranges, durationSec)
		if err != nil {
			return err
		}
	}

	segments, err := buildKeepSegmentsFromRanges(ranges, durationSec, hasDuration)
	if err != nil {
		return err
	}
	if len(segments) == 0 {
		return fmt.Errorf("silinecek aralık tüm videoyu kapsıyor")
	}

	tempDir, err := os.MkdirTemp("", "fileconverter-video-remove-*")
	if err != nil {
		return fmt.Errorf("geçici klasör oluşturulamadı: %w", err)
	}
	defer os.RemoveAll(tempDir)

	ext := filepath.Ext(output)
	if ext == "" {
		ext = filepath.Ext(input)
	}
	if ext == "" {
		ext = ".mp4"
	}

	remainingParts := make([]string, 0, len(segments))
	for i, segment := range segments {
		partPath := filepath.Join(tempDir, fmt.Sprintf("part_%02d%s", i+1, ext))
		args := []string{}
		if !verbose {
			args = append(args, "-loglevel", "error")
		}
		args = append(args, "-i", input, "-ss", formatSecondsForFFmpeg(segment.Start))
		if segment.HasEnd {
			length := segment.End - segment.Start
			if length <= 0 {
				continue
			}
			args = append(args, "-t", formatSecondsForFFmpeg(length))
		}
		args = append(args, "-c", "copy", "-y", partPath)
		if err := runFFmpegCommand(ffmpegPath, args, "video remove ara parça üretilemedi"); err != nil {
			return err
		}
		if hasContent(partPath) {
			remainingParts = append(remainingParts, partPath)
		}
	}

	if len(remainingParts) == 0 {
		return fmt.Errorf("silinecek aralık tüm videoyu kapsıyor")
	}
	if len(remainingParts) == 1 {
		singleArgs := []string{}
		if !verbose {
			singleArgs = append(singleArgs, "-loglevel", "error")
		}
		singleArgs = append(singleArgs, "-i", remainingParts[0])
		singleArgs = append(singleArgs, trimCodecArgs(codec, quality)...)
		singleArgs = append(singleArgs, converter.MetadataFFmpegArgs(metadataMode)...)
		singleArgs = append(singleArgs, "-y", output)
		return runFFmpegCommand(ffmpegPath, singleArgs, "video remove çıktı üretilemedi")
	}

	listPath := filepath.Join(tempDir, "concat.txt")
	var listBuilder strings.Builder
	for _, part := range remainingParts {
		listBuilder.WriteString(fmt.Sprintf("file '%s'\n", escapeConcatPath(part)))
	}
	listContent := listBuilder.String()
	if err := os.WriteFile(listPath, []byte(listContent), 0644); err != nil {
		return fmt.Errorf("concat listesi yazılamadı: %w", err)
	}

	concatArgs := []string{}
	if !verbose {
		concatArgs = append(concatArgs, "-loglevel", "error")
	}
	concatArgs = append(concatArgs, "-f", "concat", "-safe", "0", "-i", listPath)
	concatArgs = append(concatArgs, trimCodecArgs(codec, quality)...)
	concatArgs = append(concatArgs, converter.MetadataFFmpegArgs(metadataMode)...)
	concatArgs = append(concatArgs, "-y", output)
	return runFFmpegCommand(ffmpegPath, concatArgs, "video remove birleştirme hatası")
}

func clampTrimRangesToDuration(ranges []trimRange, durationSec float64) ([]trimRange, error) {
	const epsilon = 0.001
	clamped := make([]trimRange, 0, len(ranges))

	for _, r := range mergeTrimRanges(ranges) {
		if r.Start >= durationSec-epsilon {
			continue
		}
		if r.Start < 0 {
			r.Start = 0
		}
		if r.End > durationSec {
			r.End = durationSec
		}
		if r.End <= r.Start+epsilon {
			continue
		}
		clamped = append(clamped, r)
	}

	if len(clamped) == 0 {
		return nil, fmt.Errorf("silinecek aralıklar video süresinin dışında")
	}
	return mergeTrimRanges(clamped), nil
}

func buildKeepSegmentsFromRanges(ranges []trimRange, durationSec float64, hasDuration bool) ([]keepSegment, error) {
	const epsilon = 0.001
	if len(ranges) == 0 {
		return nil, fmt.Errorf("geçersiz aralık listesi")
	}

	segments := make([]keepSegment, 0, len(ranges)+1)
	cursor := 0.0
	for _, r := range mergeTrimRanges(ranges) {
		if r.Start > cursor+epsilon {
			segments = append(segments, keepSegment{
				Start:  cursor,
				End:    r.Start,
				HasEnd: true,
			})
		}
		if r.End > cursor {
			cursor = r.End
		}
	}

	if hasDuration {
		if cursor < durationSec-epsilon {
			segments = append(segments, keepSegment{
				Start:  cursor,
				End:    durationSec,
				HasEnd: true,
			})
		}
		return segments, nil
	}

	segments = append(segments, keepSegment{
		Start:  cursor,
		HasEnd: false,
	})
	return segments, nil
}

func trimCodecArgs(codec string, quality int) []string {
	if codec == "copy" {
		return []string{"-c", "copy"}
	}

	crf := trimCRF(quality)
	return []string{
		"-c:v", "libx264",
		"-crf", fmt.Sprintf("%d", crf),
		"-preset", "medium",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-b:a", "128k",
	}
}

func runFFmpegCommand(ffmpegPath string, args []string, prefix string) error {
	cmd := exec.Command(ffmpegPath, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s\n%s", prefix, err.Error(), string(out))
	}
	return nil
}

func formatSecondsForFFmpeg(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func hasContent(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Size() > 0
}

func escapeConcatPath(path string) string {
	return strings.ReplaceAll(path, "'", "'\\''")
}

func adjustTrimWindowByDuration(input string, startSec float64, endSec float64, mode string) (float64, float64, error) {
	durationSec, ok := probeMediaDurationSeconds(input)
	if !ok {
		return startSec, endSec, nil
	}
	return clampTrimWindowToDuration(startSec, endSec, durationSec, mode)
}

func probeMediaDurationSeconds(input string) (float64, bool) {
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		return 0, false
	}
	cmd := exec.Command(ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		input,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, false
	}
	sec, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil || sec <= 0 {
		return 0, false
	}
	return sec, true
}

func clampTrimWindowToDuration(startSec float64, endSec float64, durationSec float64, mode string) (float64, float64, error) {
	const epsilon = 0.001

	if durationSec <= 0 {
		return startSec, endSec, nil
	}
	if startSec < 0 {
		return 0, 0, fmt.Errorf("başlangıç zamanı negatif olamaz")
	}
	if startSec >= durationSec-epsilon {
		return 0, 0, fmt.Errorf("başlangıç zamanı video süresini aşıyor (%.2fs)", durationSec)
	}
	if endSec > durationSec {
		endSec = durationSec
	}
	if mode == trimModeRemove && endSec <= 0 {
		return 0, 0, fmt.Errorf("remove modunda geçerli bir bitiş zamanı gerekir")
	}
	if endSec > 0 && endSec <= startSec+epsilon {
		return 0, 0, fmt.Errorf("bitiş zamanı başlangıçtan büyük olmalıdır")
	}
	return startSec, endSec, nil
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
