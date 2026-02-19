package profile

import (
	"fmt"
	"strings"
	"time"

	"github.com/mlihgenel/fileconverter-cli/internal/batch"
	"github.com/mlihgenel/fileconverter-cli/internal/converter"
)

// Definition dönüşüm profili alanlarını tutar.
// nil pointer alanlar "profil bu alanı zorlamıyor" anlamına gelir.
type Definition struct {
	Name         string
	Quality      *int
	OnConflict   string
	Retry        *int
	RetryDelay   *time.Duration
	Report       string
	ResizePreset string
	ResizeMode   string
	Width        *float64
	Height       *float64
	Unit         string
	DPI          *float64
	MetadataMode string
}

var builtins = map[string]Definition{
	"social-story": {
		Name:         "social-story",
		Quality:      intPtr(82),
		OnConflict:   converter.ConflictVersioned,
		Retry:        intPtr(1),
		RetryDelay:   durationPtr(500 * time.Millisecond),
		Report:       batch.ReportOff,
		ResizePreset: "story",
		ResizeMode:   string(converter.ResizeModePad),
		MetadataMode: converter.MetadataStrip,
	},
	"podcast-clean": {
		Name:         "podcast-clean",
		Quality:      intPtr(90),
		OnConflict:   converter.ConflictVersioned,
		Retry:        intPtr(2),
		RetryDelay:   durationPtr(1 * time.Second),
		Report:       batch.ReportTXT,
		MetadataMode: converter.MetadataPreserve,
	},
	"archive-lossless": {
		Name:         "archive-lossless",
		Quality:      intPtr(100),
		OnConflict:   converter.ConflictVersioned,
		Retry:        intPtr(0),
		RetryDelay:   durationPtr(0),
		Report:       batch.ReportJSON,
		MetadataMode: converter.MetadataPreserve,
	},
}

// Resolve isimden profile döner.
func Resolve(name string) (Definition, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return Definition{}, fmt.Errorf("profil adi bos")
	}
	p, ok := builtins[key]
	if !ok {
		return Definition{}, fmt.Errorf("profil bulunamadi: %s", name)
	}
	return p, nil
}

// Names built-in profil isimlerini döner.
func Names() []string {
	return []string{"social-story", "podcast-clean", "archive-lossless"}
}

func intPtr(v int) *int { return &v }

func floatPtr(v float64) *float64 { return &v }

func durationPtr(v time.Duration) *time.Duration { return &v }

// Helper exportları test/ileriki genişleme için tutuldu.
var (
	IntPtr      = intPtr
	FloatPtr    = floatPtr
	DurationPtr = durationPtr
)
