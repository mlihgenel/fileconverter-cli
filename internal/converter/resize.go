package converter

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultResizeUnit = "px"
	defaultResizeDPI  = 96.0
)

// ResizeMode hedef boyutlandırma davranışını belirler.
type ResizeMode string

const (
	ResizeModePad     ResizeMode = "pad"
	ResizeModeFit     ResizeMode = "fit"
	ResizeModeFill    ResizeMode = "fill"
	ResizeModeStretch ResizeMode = "stretch"
)

// ResizeSpec tek bir dönüşüm için boyutlandırma ayarlarını tutar.
type ResizeSpec struct {
	Width  int
	Height int
	Mode   ResizeMode
	Unit   string
	DPI    float64
	Preset string
}

// ResizePreset hazır boyut profili.
type ResizePreset struct {
	Name        string
	Width       int
	Height      int
	Description string
}

var resizePresetCatalog = []ResizePreset{
	{Name: "story", Width: 1080, Height: 1920, Description: "Story / Reels / TikTok (9:16)"},
	{Name: "vertical-hd", Width: 720, Height: 1280, Description: "Dikey HD (9:16)"},
	{Name: "vertical-fullhd", Width: 1080, Height: 1920, Description: "Dikey Full HD (9:16)"},
	{Name: "instagram-portrait", Width: 1080, Height: 1350, Description: "Instagram Portre (4:5)"},
	{Name: "square", Width: 1080, Height: 1080, Description: "Kare (1:1)"},
	{Name: "hd", Width: 1280, Height: 720, Description: "HD (16:9)"},
	{Name: "fullhd", Width: 1920, Height: 1080, Description: "Full HD (16:9)"},
	{Name: "2k", Width: 2560, Height: 1440, Description: "2K QHD (16:9)"},
	{Name: "4k", Width: 3840, Height: 2160, Description: "4K UHD (16:9)"},
}

var resizePresetAliases = map[string]string{
	"reel":            "story",
	"reels":           "story",
	"tiktok":          "story",
	"shorts":          "story",
	"youtube-shorts":  "story",
	"instagram-story": "story",
	"portrait":        "instagram-portrait",
	"1:1":             "square",
	"4:5":             "instagram-portrait",
	"720p":            "hd",
	"1080p":           "fullhd",
	"uhd":             "4k",
}

var resizableFormatSet = map[string]bool{
	"png":  true,
	"jpg":  true,
	"webp": true,
	"bmp":  true,
	"gif":  true,
	"tif":  true,
	"ico":  true,
	"mp4":  true,
	"mov":  true,
	"mkv":  true,
	"avi":  true,
	"webm": true,
	"m4v":  true,
	"wmv":  true,
	"flv":  true,
}

// IsResizableFormat formatın boyutlandırma desteği olup olmadığını döner.
func IsResizableFormat(format string) bool {
	return resizableFormatSet[NormalizeFormat(format)]
}

// ParseResizeMode kullanıcı girişinden boyutlandırma modunu üretir.
func ParseResizeMode(raw string) (ResizeMode, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "pad", "contain", "letterbox":
		return ResizeModePad, nil
	case "fit", "inside":
		return ResizeModeFit, nil
	case "fill", "crop", "cover":
		return ResizeModeFill, nil
	case "stretch", "distort":
		return ResizeModeStretch, nil
	default:
		return "", fmt.Errorf("geçersiz resize modu: %s (geçerli: pad, fit, fill, stretch)", raw)
	}
}

// BuildResizeSpec bayraklardan ortak boyutlandırma ayarını üretir.
func BuildResizeSpec(preset string, width float64, height float64, unit string, mode string, dpi float64) (*ResizeSpec, error) {
	hasPreset := strings.TrimSpace(preset) != ""
	hasManual := width > 0 || height > 0

	if !hasPreset && !hasManual {
		return nil, nil
	}
	if hasPreset && hasManual {
		return nil, fmt.Errorf("boyutlandırmada preset ve manuel ölçü aynı anda kullanılamaz")
	}

	resizeMode, err := ParseResizeMode(mode)
	if err != nil {
		return nil, err
	}

	if hasPreset {
		p, ok := ResolveResizePreset(preset)
		if !ok {
			return nil, fmt.Errorf("bilinmeyen preset: %s (örnek: %s)", preset, strings.Join(ResizePresetNames(), ", "))
		}
		return &ResizeSpec{
			Width:  p.Width,
			Height: p.Height,
			Mode:   resizeMode,
			Unit:   defaultResizeUnit,
			DPI:    defaultResizeDPI,
			Preset: p.Name,
		}, nil
	}

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("manuel boyutlandırma için --width ve --height birlikte verilmelidir")
	}

	normalizedUnit, err := normalizeResizeUnit(unit)
	if err != nil {
		return nil, err
	}

	if dpi <= 0 {
		dpi = defaultResizeDPI
	}

	wpx, err := dimensionToPixels(width, normalizedUnit, dpi)
	if err != nil {
		return nil, fmt.Errorf("width hatası: %w", err)
	}
	hpx, err := dimensionToPixels(height, normalizedUnit, dpi)
	if err != nil {
		return nil, fmt.Errorf("height hatası: %w", err)
	}

	return &ResizeSpec{
		Width:  wpx,
		Height: hpx,
		Mode:   resizeMode,
		Unit:   normalizedUnit,
		DPI:    dpi,
	}, nil
}

// ResolveResizePreset preset adından çözümleme yapar.
func ResolveResizePreset(raw string) (ResizePreset, bool) {
	normalized := normalizePresetName(raw)
	if normalized == "" {
		return ResizePreset{}, false
	}

	if alias, ok := resizePresetAliases[normalized]; ok {
		normalized = alias
	}

	if w, h, ok := parseDimensionPair(normalized); ok {
		return ResizePreset{
			Name:        fmt.Sprintf("%dx%d", w, h),
			Width:       w,
			Height:      h,
			Description: "Özel ölçü",
		}, true
	}

	for _, p := range resizePresetCatalog {
		if p.Name == normalized {
			return p, true
		}
	}
	return ResizePreset{}, false
}

// ResizePresets hazır preset listesini döner.
func ResizePresets() []ResizePreset {
	result := make([]ResizePreset, len(resizePresetCatalog))
	copy(result, resizePresetCatalog)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// ResizePresetNames preset adlarını alfabetik döner.
func ResizePresetNames() []string {
	presets := ResizePresets()
	names := make([]string, 0, len(presets))
	for _, p := range presets {
		names = append(names, p.Name)
	}
	return names
}

func normalizeResizeUnit(raw string) (string, error) {
	unit := strings.ToLower(strings.TrimSpace(raw))
	switch unit {
	case "", "px", "pixel", "pixels":
		return "px", nil
	case "cm", "centimeter", "centimeters":
		return "cm", nil
	default:
		return "", fmt.Errorf("geçersiz birim: %s (geçerli: px, cm)", raw)
	}
}

func dimensionToPixels(value float64, unit string, dpi float64) (int, error) {
	if value <= 0 {
		return 0, fmt.Errorf("değer 0'dan büyük olmalı")
	}

	pixels := value
	if unit == "cm" {
		pixels = (value / 2.54) * dpi
	}

	rounded := int(math.Round(pixels))
	if rounded < 1 {
		return 0, fmt.Errorf("piksel karşılığı en az 1 olmalı")
	}
	return rounded, nil
}

func normalizePresetName(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, "×", "x")
	return s
}

func parseDimensionPair(raw string) (int, int, bool) {
	candidate := strings.ReplaceAll(strings.TrimSpace(raw), " ", "")
	candidate = strings.ReplaceAll(candidate, "×", "x")
	parts := strings.Split(candidate, "x")
	if len(parts) != 2 {
		return 0, 0, false
	}

	w, err1 := strconv.Atoi(parts[0])
	h, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		return 0, 0, false
	}
	return w, h, true
}
