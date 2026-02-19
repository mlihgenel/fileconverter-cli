package converter

import "strings"

const (
	MetadataAuto     = "auto"
	MetadataPreserve = "preserve"
	MetadataStrip    = "strip"
)

// NormalizeMetadataMode metadata modunu normalize eder.
func NormalizeMetadataMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", MetadataAuto:
		return MetadataAuto
	case MetadataPreserve:
		return MetadataPreserve
	case MetadataStrip:
		return MetadataStrip
	default:
		return ""
	}
}

// MetadataFFmpegArgs metadata moduna göre FFmpeg argümanlarını döner.
func MetadataFFmpegArgs(mode string) []string {
	switch NormalizeMetadataMode(mode) {
	case MetadataStrip:
		return []string{"-map_metadata", "-1"}
	case MetadataPreserve, MetadataAuto:
		return nil
	default:
		return nil
	}
}
