package converter

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// Options dönüşüm seçeneklerini tutar
type Options struct {
	Quality int    // 1-100 arası kalite ayarı
	Verbose bool   // Detaylı çıktı modu
	Name    string // Çıktı dosya adı (opsiyonel)
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
	ext := filepath.Ext(filename)
	return NormalizeFormat(ext)
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
