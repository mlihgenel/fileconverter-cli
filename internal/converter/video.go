package converter

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// VideoConverter video dosyalarını FFmpeg ile dönüştürür.
type VideoConverter struct {
	ffmpegPath string
}

func init() {
	Register(&VideoConverter{})
}

func (v *VideoConverter) Name() string {
	return "Video Converter (FFmpeg)"
}

// videoInputFormats kaynak olarak desteklenen video formatları.
var videoInputFormats = []string{"mp4", "mov", "mkv", "avi", "webm", "m4v", "wmv", "flv"}

// videoOutputFormats hedef olarak desteklenen video/gif formatları.
var videoOutputFormats = []string{"mp4", "mov", "mkv", "avi", "webm", "m4v", "wmv", "flv", "gif"}

func (v *VideoConverter) SupportedConversions() []ConversionPair {
	var pairs []ConversionPair
	for _, from := range videoInputFormats {
		for _, to := range videoOutputFormats {
			if from == to {
				continue
			}
			pairs = append(pairs, ConversionPair{
				From:        from,
				To:          to,
				Description: fmt.Sprintf("%s → %s", strings.ToUpper(from), strings.ToUpper(to)),
			})
		}
	}
	return pairs
}

func (v *VideoConverter) SupportsConversion(from, to string) bool {
	from = NormalizeFormat(from)
	to = NormalizeFormat(to)

	fromSupported := false
	toSupported := false

	for _, f := range videoInputFormats {
		if f == from {
			fromSupported = true
			break
		}
	}
	for _, f := range videoOutputFormats {
		if f == to {
			toSupported = true
			break
		}
	}

	return fromSupported && toSupported
}

func (v *VideoConverter) Convert(input string, output string, opts Options) error {
	ffmpegPath, err := v.findFFmpeg()
	if err != nil {
		return err
	}

	to := DetectFormat(output)
	args := []string{}

	if !opts.Verbose {
		args = append(args, "-loglevel", "error")
	}

	args = append(args, "-i", input, "-y")

	filters := make([]string, 0, 2)
	if to == "gif" {
		fps, width := gifProfile(opts.Quality)
		if opts.Resize == nil {
			filters = append(filters, fmt.Sprintf("fps=%d,scale=%d:-1:flags=lanczos", fps, width))
		} else {
			filters = append(filters, fmt.Sprintf("fps=%d", fps))
		}
	}

	if opts.Resize != nil {
		resizeFilter, err := buildVideoResizeFilter(*opts.Resize)
		if err != nil {
			return err
		}
		filters = append(filters, resizeFilter)
		args = append(args, "-sws_flags", "lanczos+accurate_rnd")
	}

	if len(filters) > 0 {
		args = append(args, "-vf", strings.Join(filters, ","))
	}

	// Video çıktılarında varsa sesi koru, yoksa sessiz devam et.
	if to != "gif" {
		args = append(args, "-map", "0:v:0", "-map", "0:a?")
	}

	args = append(args, v.getCodecArgs(to, opts.Quality)...)
	args = append(args, MetadataFFmpegArgs(opts.MetadataMode)...)
	args = append(args, output)

	cmd := exec.Command(ffmpegPath, args...)
	if outputBytes, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("FFmpeg hatası: %s\n%s", err.Error(), string(outputBytes))
	}

	return nil
}

// getCodecArgs hedef format ve kaliteye göre FFmpeg parametrelerini döner.
func (v *VideoConverter) getCodecArgs(to string, quality int) []string {
	crf := videoCRF(quality)

	switch to {
	case "gif":
		return []string{"-loop", "0", "-an"}
	case "webm":
		webmCRF := crf + 6
		if webmCRF > 40 {
			webmCRF = 40
		}
		return []string{
			"-c:v", "libvpx-vp9",
			"-crf", strconv.Itoa(webmCRF),
			"-b:v", "0",
			"-row-mt", "1",
			"-c:a", "libopus",
			"-b:a", "128k",
		}
	case "avi":
		return []string{
			"-c:v", "mpeg4",
			"-q:v", strconv.Itoa(videoQScale(quality)),
			"-c:a", "mp3",
			"-b:a", "192k",
		}
	case "wmv":
		return []string{
			"-c:v", "wmv2",
			"-c:a", "wmav2",
		}
	case "flv":
		return []string{
			"-c:v", "flv",
			"-c:a", "mp3",
			"-ar", "44100",
		}
	case "mp4", "m4v", "mov":
		return []string{
			"-c:v", "libx264",
			"-crf", strconv.Itoa(crf),
			"-preset", "medium",
			"-pix_fmt", "yuv420p",
			"-movflags", "+faststart",
			"-c:a", "aac",
			"-b:a", "128k",
		}
	default: // mkv ve gelecekteki h264 uyumlu kapsayıcılar
		return []string{
			"-c:v", "libx264",
			"-crf", strconv.Itoa(crf),
			"-preset", "medium",
			"-pix_fmt", "yuv420p",
			"-c:a", "aac",
			"-b:a", "128k",
		}
	}
}

func buildVideoResizeFilter(spec ResizeSpec) (string, error) {
	width := normalizeVideoDimension(spec.Width)
	height := normalizeVideoDimension(spec.Height)
	if width <= 0 || height <= 0 {
		return "", fmt.Errorf("geçersiz video hedef boyutu: %dx%d", spec.Width, spec.Height)
	}

	switch spec.Mode {
	case ResizeModeStretch:
		return fmt.Sprintf("scale=%d:%d:flags=lanczos", width, height), nil
	case ResizeModeFit:
		return fmt.Sprintf("scale=%d:%d:flags=lanczos:force_original_aspect_ratio=decrease:force_divisible_by=2", width, height), nil
	case ResizeModeFill:
		return fmt.Sprintf("scale=%d:%d:flags=lanczos:force_original_aspect_ratio=increase:force_divisible_by=2,crop=%d:%d", width, height, width, height), nil
	case ResizeModePad:
		return fmt.Sprintf("scale=%d:%d:flags=lanczos:force_original_aspect_ratio=decrease:force_divisible_by=2,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black", width, height, width, height), nil
	default:
		return "", fmt.Errorf("desteklenmeyen video resize modu: %s", spec.Mode)
	}
}

func normalizeVideoDimension(value int) int {
	if value < 2 {
		return 2
	}
	if value%2 == 1 {
		return value + 1
	}
	return value
}

func videoCRF(quality int) int {
	// FFmpeg CRF için mantıklı varsayılan.
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

func videoQScale(quality int) int {
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

func gifProfile(quality int) (fps int, width int) {
	if quality <= 0 {
		return 12, 800
	}
	switch {
	case quality <= 25:
		return 8, 480
	case quality <= 50:
		return 10, 640
	case quality <= 75:
		return 12, 800
	default:
		return 15, 960
	}
}

// findFFmpeg sistemde FFmpeg'i arar.
func (v *VideoConverter) findFFmpeg() (string, error) {
	if v.ffmpegPath != "" {
		return v.ffmpegPath, nil
	}

	if envPath := os.Getenv("FFMPEG_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			v.ffmpegPath = envPath
			return envPath, nil
		}
	}

	paths := []string{"ffmpeg"}
	if runtime.GOOS == "darwin" {
		paths = append(paths, "/opt/homebrew/bin/ffmpeg", "/usr/local/bin/ffmpeg")
	} else if runtime.GOOS == "linux" {
		paths = append(paths, "/usr/bin/ffmpeg", "/usr/local/bin/ffmpeg")
	}

	for _, p := range paths {
		if path, err := exec.LookPath(p); err == nil {
			v.ffmpegPath = path
			return path, nil
		}
	}

	return "", fmt.Errorf(
		"FFmpeg bulunamadı! Video dönüşümü için FFmpeg kurulu olmalıdır.\n\n" +
			"Kurulum:\n" +
			"  macOS:   brew install ffmpeg\n" +
			"  Ubuntu:  sudo apt install ffmpeg\n" +
			"  Windows: https://ffmpeg.org/download.html\n")
}
