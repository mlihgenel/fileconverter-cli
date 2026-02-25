package converter

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// FileInfo dosya hakkındaki tüm bilgileri tutar
type FileInfo struct {
	Path     string `json:"path"`
	FileName string `json:"file_name"`
	Format   string `json:"format"`
	Category string `json:"category"` // "image", "video", "audio", "document"
	Size     int64  `json:"size_bytes"`
	SizeText string `json:"size_text"`

	// Görsel
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`

	// Video / Ses (FFprobe)
	Duration   string  `json:"duration,omitempty"`
	VideoCodec string  `json:"video_codec,omitempty"`
	AudioCodec string  `json:"audio_codec,omitempty"`
	Bitrate    string  `json:"bitrate,omitempty"`
	FPS        float64 `json:"fps,omitempty"`
	Channels   int     `json:"channels,omitempty"`
	SampleRate int     `json:"sample_rate,omitempty"`
	Resolution string  `json:"resolution,omitempty"`
}

// categorizeFormat format adından kategori belirler
func categorizeFormat(format string) string {
	imageFormatsSet := map[string]bool{
		"png": true, "jpg": true, "webp": true, "bmp": true,
		"gif": true, "tif": true, "ico": true, "svg": true,
		"heic": true, "heif": true,
	}
	videoFormatsSet := map[string]bool{
		"mp4": true, "mov": true, "mkv": true, "avi": true,
		"webm": true, "m4v": true, "wmv": true, "flv": true,
	}
	audioFormatsSet := map[string]bool{
		"mp3": true, "wav": true, "ogg": true, "flac": true,
		"aac": true, "m4a": true, "wma": true, "opus": true,
	}
	docFormatsSet := map[string]bool{
		"md": true, "html": true, "pdf": true, "docx": true,
		"txt": true, "odt": true, "rtf": true, "csv": true,
	}

	if imageFormatsSet[format] {
		return "image"
	}
	if videoFormatsSet[format] {
		return "video"
	}
	if audioFormatsSet[format] {
		return "audio"
	}
	if docFormatsSet[format] {
		return "document"
	}
	return "unknown"
}

// GetFileInfo dosya hakkında bilgi toplar
func GetFileInfo(path string) (FileInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, fmt.Errorf("dosya bulunamadı: %w", err)
	}

	format := DetectFormat(path)
	category := categorizeFormat(format)

	info := FileInfo{
		Path:     path,
		FileName: filepath.Base(path),
		Format:   strings.ToUpper(format),
		Category: category,
		Size:     stat.Size(),
		SizeText: formatInfoSize(stat.Size()),
	}

	switch category {
	case "image":
		fillImageInfo(&info, path)
	case "video", "audio":
		fillMediaInfo(&info, path)
	}

	return info, nil
}

// fillImageInfo Go image.DecodeConfig ile görsel boyutlarını okur
func fillImageInfo(info *FileInfo, path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return
	}
	info.Width = cfg.Width
	info.Height = cfg.Height
	info.Resolution = fmt.Sprintf("%dx%d", cfg.Width, cfg.Height)
}

// ffprobeResult ffprobe JSON çıktısının ilgili alanları
type ffprobeResult struct {
	Format struct {
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
	} `json:"format"`
	Streams []struct {
		CodecType  string `json:"codec_type"`
		CodecName  string `json:"codec_name"`
		Width      int    `json:"width,omitempty"`
		Height     int    `json:"height,omitempty"`
		RFrameRate string `json:"r_frame_rate,omitempty"`
		Channels   int    `json:"channels,omitempty"`
		SampleRate string `json:"sample_rate,omitempty"`
	} `json:"streams"`
}

// fillMediaInfo FFprobe ile video/ses bilgilerini okur
func fillMediaInfo(info *FileInfo, path string) {
	ffprobePath := findFFprobe()
	if ffprobePath == "" {
		return
	}

	cmd := exec.Command(ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)
	output, err := cmd.Output()
	if err != nil {
		return
	}

	var result ffprobeResult
	if err := json.Unmarshal(output, &result); err != nil {
		return
	}

	// Duration
	if result.Format.Duration != "" {
		if dur, err := strconv.ParseFloat(result.Format.Duration, 64); err == nil {
			hours := int(dur) / 3600
			minutes := (int(dur) % 3600) / 60
			seconds := int(dur) % 60
			if hours > 0 {
				info.Duration = fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
			} else {
				info.Duration = fmt.Sprintf("%02d:%02d", minutes, seconds)
			}
		}
	}

	// Bitrate
	if result.Format.BitRate != "" {
		if br, err := strconv.ParseInt(result.Format.BitRate, 10, 64); err == nil {
			info.Bitrate = fmt.Sprintf("%d kbps", br/1000)
		}
	}

	// Streams
	for _, s := range result.Streams {
		switch s.CodecType {
		case "video":
			info.VideoCodec = s.CodecName
			if s.Width > 0 && s.Height > 0 {
				info.Width = s.Width
				info.Height = s.Height
				info.Resolution = fmt.Sprintf("%dx%d", s.Width, s.Height)
			}
			if s.RFrameRate != "" {
				info.FPS = parseFrameRate(s.RFrameRate)
			}
		case "audio":
			info.AudioCodec = s.CodecName
			info.Channels = s.Channels
			if s.SampleRate != "" {
				if sr, err := strconv.Atoi(s.SampleRate); err == nil {
					info.SampleRate = sr
				}
			}
		}
	}
}

// parseFrameRate "30000/1001" gibi kare oranlarını float'a çevirir
func parseFrameRate(rate string) float64 {
	parts := strings.SplitN(rate, "/", 2)
	if len(parts) == 2 {
		num, err1 := strconv.ParseFloat(parts[0], 64)
		den, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 == nil && err2 == nil && den != 0 {
			return num / den
		}
	}
	if f, err := strconv.ParseFloat(rate, 64); err == nil {
		return f
	}
	return 0
}

// findFFprobe sistemde ffprobe'u arar
func findFFprobe() string {
	paths := []string{"ffprobe"}
	if runtime.GOOS == "darwin" {
		paths = append(paths, "/opt/homebrew/bin/ffprobe", "/usr/local/bin/ffprobe")
	} else if runtime.GOOS == "linux" {
		paths = append(paths, "/usr/bin/ffprobe", "/usr/local/bin/ffprobe")
	}

	for _, p := range paths {
		if path, err := exec.LookPath(p); err == nil {
			return path
		}
	}
	return ""
}

// formatInfoSize dosya boyutunu okunabilir hale getirir
func formatInfoSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
