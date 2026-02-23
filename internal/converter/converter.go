package converter

import (
	"archive/zip"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// Options dönüşüm seçeneklerini tutar
type Options struct {
	Quality int    // 1-100 arası kalite ayarı
	Verbose bool   // Detaylı çıktı modu
	Name    string // Çıktı dosya adı (opsiyonel)
	Resize  *ResizeSpec
	// MetadataMode: auto, preserve, strip
	MetadataMode string
}

// Result dönüşüm sonucunu tutar
type Result struct {
	InputFile  string
	OutputFile string
	Success    bool
	Error      error
}

// Converter arayüzü — tüm dönüştürücüler bunu implemente eder
type Converter interface {
	// Convert dosyayı dönüştürür
	Convert(input string, output string, opts Options) error
	// SupportsConversion bu dönüşümü destekleyip desteklemediğini kontrol eder
	SupportsConversion(from, to string) bool
	// Name dönüştürücünün adını döner
	Name() string
	// SupportedConversions desteklenen tüm dönüşüm çiftlerini döner
	SupportedConversions() []ConversionPair
}

// ConversionPair bir kaynak-hedef format çiftini temsil eder
type ConversionPair struct {
	From        string
	To          string
	Description string
}

// Registry dönüştürücüleri yöneten merkezi kayıt sistemi
type Registry struct {
	converters []Converter
	mu         sync.RWMutex
}

// globalRegistry uygulama genelinde tek bir registry
var globalRegistry = &Registry{}

// Register yeni bir converter kaydeder
func Register(c Converter) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.converters = append(globalRegistry.converters, c)
}

// FindConverter verilen format çifti için uygun converter'ı bulur
func FindConverter(from, to string) (Converter, error) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	from = NormalizeFormat(from)
	to = NormalizeFormat(to)

	for _, c := range globalRegistry.converters {
		if c.SupportsConversion(from, to) {
			return c, nil
		}
	}
	return nil, fmt.Errorf("'%s' → '%s' dönüşümü desteklenmiyor", from, to)
}

// GetAllConversions tüm desteklenen dönüşümleri döner
func GetAllConversions() []ConversionPair {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	var all []ConversionPair
	for _, c := range globalRegistry.converters {
		all = append(all, c.SupportedConversions()...)
	}
	return all
}

// GetConversionsFrom belirli bir formattan yapılabilecek dönüşümleri döner
func GetConversionsFrom(from string) []ConversionPair {
	from = NormalizeFormat(from)
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	var results []ConversionPair
	for _, c := range globalRegistry.converters {
		for _, pair := range c.SupportedConversions() {
			if pair.From == from {
				results = append(results, pair)
			}
		}
	}
	return results
}

// GetConversionsTo belirli bir formata yapılabilecek dönüşümleri döner
func GetConversionsTo(to string) []ConversionPair {
	to = NormalizeFormat(to)
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	var results []ConversionPair
	for _, c := range globalRegistry.converters {
		for _, pair := range c.SupportedConversions() {
			if pair.To == to {
				results = append(results, pair)
			}
		}
	}
	return results
}

// NormalizeFormat format adını standartlaştırır (.md → md, .JPEG → jpeg vb.)
func NormalizeFormat(format string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	format = strings.TrimPrefix(format, ".")

	// Yaygın alternatif isimler
	aliases := map[string]string{
		"markdown":       "md",
		"jpeg":           "jpg",
		"tiff":           "tif",
		"wave":           "wav",
		"text":           "txt",
		"plaintext":      "txt",
		"opendocument":   "odt",
		"richtextformat": "rtf",
	}

	if alias, ok := aliases[format]; ok {
		return alias
	}
	return format
}

// DetectFormat dosya uzantısından format algılar
func DetectFormat(filename string) string {
	ext := NormalizeFormat(filepath.Ext(filename))
	detected := detectFormatFromContent(filename)

	switch {
	case detected == "":
		return ext
	case ext == "":
		return detected
	case ext == detected:
		return ext
	}

	// MIME tabanlı zayıf tespitlerde bilinen uzantıyı koru.
	if isWeakContentGuess(detected) && ext != "" && !isTextLikeFormat(ext) {
		return ext
	}

	// Metin tabanlı içeriklerde uzantı genelde daha anlamlıdır (örn: csv vs txt).
	if isTextLikeFormat(ext) && isTextLikeFormat(detected) {
		return ext
	}

	return detected
}

func detectFormatFromContent(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	header := make([]byte, 8192)
	n, readErr := f.Read(header)
	if readErr != nil && n == 0 {
		return ""
	}
	header = header[:n]
	if len(header) == 0 {
		return ""
	}

	if byMagic := detectFormatByMagic(header); byMagic != "" {
		return byMagic
	}
	if byMIME := detectFormatByMIME(header); byMIME != "" {
		return byMIME
	}
	if isZipHeader(header) {
		if byZip := detectZipDocumentFormat(path); byZip != "" {
			return byZip
		}
	}
	return ""
}

func detectFormatByMagic(header []byte) string {
	if len(header) >= 12 && string(header[0:4]) == "RIFF" {
		switch string(header[8:12]) {
		case "WAVE":
			return "wav"
		case "AVI ":
			return "avi"
		case "WEBP":
			return "webp"
		}
	}

	if len(header) >= 4 {
		switch string(header[0:4]) {
		case "%PDF":
			return "pdf"
		case "fLaC":
			return "flac"
		case "OggS":
			return "ogg"
		case "\x1A\x45\xDF\xA3":
			// Matroska/WebM ayrımı için DocType alanını kontrol et.
			if strings.Contains(string(header), "webm") {
				return "webm"
			}
			return "mkv"
		}
	}

	if len(header) >= 3 && string(header[0:3]) == "ID3" {
		return "mp3"
	}
	if len(header) >= 2 && header[0] == 0xFF && (header[1]&0xE0) == 0xE0 {
		return "mp3"
	}

	if len(header) >= 3 && string(header[0:3]) == "FLV" {
		return "flv"
	}

	if len(header) >= 16 {
		asf := []byte{0x30, 0x26, 0xB2, 0x75, 0x8E, 0x66, 0xCF, 0x11, 0xA6, 0xD9, 0x00, 0xAA, 0x00, 0x62, 0xCE, 0x6C}
		if slices.Equal(header[:16], asf) {
			return "wmv"
		}
	}

	if len(header) >= 12 && string(header[4:8]) == "ftyp" {
		brand := string(header[8:12])
		switch brand {
		case "M4A ", "M4B ", "M4P ":
			return "m4a"
		case "M4V ":
			return "m4v"
		case "qt  ":
			return "mov"
		default:
			return "mp4"
		}
	}

	return ""
}

func detectFormatByMIME(header []byte) string {
	switch http.DetectContentType(header) {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	case "image/bmp":
		return "bmp"
	case "image/tiff":
		return "tif"
	case "image/x-icon", "image/vnd.microsoft.icon":
		return "ico"
	case "application/pdf":
		return "pdf"
	case "text/html; charset=utf-8":
		return "html"
	case "text/plain; charset=utf-8":
		return "txt"
	case "audio/mpeg":
		return "mp3"
	case "audio/ogg":
		return "ogg"
	case "video/mp4":
		return "mp4"
	case "video/webm":
		return "webm"
	}
	return ""
}

func detectZipDocumentFormat(path string) string {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return ""
	}
	defer zr.Close()

	for _, f := range zr.File {
		name := strings.ToLower(f.Name)
		switch {
		case strings.HasPrefix(name, "word/"):
			return "docx"
		case name == "mimetype":
			rc, err := f.Open()
			if err != nil {
				continue
			}
			buf := make([]byte, 128)
			n, _ := rc.Read(buf)
			_ = rc.Close()
			if strings.Contains(string(buf[:n]), "application/vnd.oasis.opendocument.text") {
				return "odt"
			}
		}
	}
	return ""
}

func isZipHeader(header []byte) bool {
	if len(header) < 4 {
		return false
	}
	sig := string(header[:4])
	return sig == "PK\x03\x04" || sig == "PK\x05\x06" || sig == "PK\x07\x08"
}

func isTextLikeFormat(format string) bool {
	switch NormalizeFormat(format) {
	case "txt", "md", "csv", "html", "rtf":
		return true
	default:
		return false
	}
}

func isWeakContentGuess(format string) bool {
	switch NormalizeFormat(format) {
	case "txt", "html":
		return true
	default:
		return false
	}
}

// FormatExtensions verilen format için eşdeğer dosya uzantılarını döner.
func FormatExtensions(format string) []string {
	switch NormalizeFormat(format) {
	case "jpg":
		return []string{".jpg", ".jpeg"}
	case "tif":
		return []string{".tif", ".tiff"}
	default:
		n := NormalizeFormat(format)
		if n == "" {
			return nil
		}
		return []string{"." + n}
	}
}

// HasFormatExtension dosyanın uzantısının verilen format ile eşleşip eşleşmediğini kontrol eder.
func HasFormatExtension(path string, format string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return false
	}
	return slices.Contains(FormatExtensions(format), ext)
}

// FormatFilterLabel formatın dosya uzantısı etiketini döner (ör. "jpg/jpeg").
func FormatFilterLabel(format string) string {
	exts := FormatExtensions(format)
	if len(exts) == 0 {
		return NormalizeFormat(format)
	}
	labels := make([]string, 0, len(exts))
	for _, ext := range exts {
		labels = append(labels, strings.TrimPrefix(ext, "."))
	}
	return strings.Join(labels, "/")
}

// BuildOutputPath çıktı dosya yolunu oluşturur
func BuildOutputPath(inputPath, outputDir, targetFormat, customName string) string {
	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))

	if customName != "" {
		baseName = customName
	}

	outputFile := baseName + "." + targetFormat

	if outputDir != "" {
		return filepath.Join(outputDir, outputFile)
	}

	return filepath.Join(filepath.Dir(inputPath), outputFile)
}

// GetAllFormats tüm benzersiz formatları döner
func GetAllFormats() []string {
	pairs := GetAllConversions()
	formatSet := make(map[string]bool)
	for _, p := range pairs {
		formatSet[p.From] = true
		formatSet[p.To] = true
	}

	var formats []string
	for f := range formatSet {
		formats = append(formats, f)
	}
	return formats
}
