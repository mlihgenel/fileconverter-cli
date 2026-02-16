package converter

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// AudioConverter ses dosyalarını FFmpeg ile dönüştürür
type AudioConverter struct {
	ffmpegPath string
}

func init() {
	Register(&AudioConverter{})
}

func (a *AudioConverter) Name() string {
	return "Audio Converter (FFmpeg)"
}

// audioFormats desteklenen ses formatları
var audioFormats = []string{"mp3", "wav", "ogg", "flac", "aac", "m4a", "wma", "opus", "webm"}

func (a *AudioConverter) SupportedConversions() []ConversionPair {
	var pairs []ConversionPair
	for _, from := range audioFormats {
		for _, to := range audioFormats {
			if from != to {
				pairs = append(pairs, ConversionPair{
					From:        from,
					To:          to,
					Description: fmt.Sprintf("%s → %s", strings.ToUpper(from), strings.ToUpper(to)),
				})
			}
		}
	}
	return pairs
}

func (a *AudioConverter) SupportsConversion(from, to string) bool {
	fromSupported := false
	toSupported := false
	for _, f := range audioFormats {
		if f == from {
			fromSupported = true
		}
		if f == to {
			toSupported = true
		}
	}
	return fromSupported && toSupported && from != to
}

func (a *AudioConverter) Convert(input string, output string, opts Options) error {
	ffmpegPath, err := a.findFFmpeg()
	if err != nil {
		return err
	}

	to := DetectFormat(output)
	args := []string{"-i", input, "-y"} // -y: overwrite

	// Codec ve kalite ayarları
	args = append(args, a.getCodecArgs(to, opts.Quality)...)

	// Verbose değilse sessiz mod
	if !opts.Verbose {
		args = append([]string{"-loglevel", "error"}, args[0:]...)
		// Rebuild: loglevel should come before -i
		args = []string{"-loglevel", "error", "-i", input, "-y"}
		args = append(args, a.getCodecArgs(to, opts.Quality)...)
	}

	args = append(args, output)

	cmd := exec.Command(ffmpegPath, args...)
	if outputBytes, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("FFmpeg hatası: %s\n%s", err.Error(), string(outputBytes))
	}

	return nil
}

// getCodecArgs hedef format ve kaliteye göre FFmpeg argümanlarını döner
func (a *AudioConverter) getCodecArgs(to string, quality int) []string {
	// Varsayılan bitrate belirle
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

	switch to {
	case "mp3":
		return []string{"-codec:a", "libmp3lame", "-b:a", bitrate}
	case "wav":
		return []string{"-codec:a", "pcm_s16le"} // WAV kalitesiz — lossless
	case "ogg":
		return []string{"-codec:a", "libvorbis", "-b:a", bitrate}
	case "flac":
		return []string{"-codec:a", "flac"} // FLAC lossless
	case "aac":
		return []string{"-codec:a", "aac", "-b:a", bitrate}
	case "m4a":
		return []string{"-codec:a", "aac", "-b:a", bitrate}
	case "wma":
		return []string{"-codec:a", "wmav2", "-b:a", bitrate}
	default:
		return []string{"-b:a", bitrate}
	}
}

// findFFmpeg sistemde FFmpeg'i arar
func (a *AudioConverter) findFFmpeg() (string, error) {
	if a.ffmpegPath != "" {
		return a.ffmpegPath, nil
	}

	// Yaygın FFmpeg konumları
	paths := []string{"ffmpeg"}

	if runtime.GOOS == "darwin" {
		paths = append(paths, "/opt/homebrew/bin/ffmpeg", "/usr/local/bin/ffmpeg")
	} else if runtime.GOOS == "linux" {
		paths = append(paths, "/usr/bin/ffmpeg", "/usr/local/bin/ffmpeg")
	}

	for _, p := range paths {
		if path, err := exec.LookPath(p); err == nil {
			a.ffmpegPath = path
			return path, nil
		}
	}

	return "", fmt.Errorf(
		"FFmpeg bulunamadı! Ses dönüşümü için FFmpeg kurulu olmalıdır.\n\n" +
			"Kurulum:\n" +
			"  macOS:   brew install ffmpeg\n" +
			"  Ubuntu:  sudo apt install ffmpeg\n" +
			"  Windows: https://ffmpeg.org/download.html\n")
}

// IsFFmpegAvailable FFmpeg'in kurulu olup olmadığını kontrol eder
func IsFFmpegAvailable() bool {
	ac := &AudioConverter{}
	_, err := ac.findFFmpeg()
	return err == nil
}
