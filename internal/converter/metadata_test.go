package converter

import (
	"reflect"
	"testing"
)

func TestNormalizeMetadataMode(t *testing.T) {
	if got := NormalizeMetadataMode(""); got != MetadataAuto {
		t.Fatalf("expected auto, got %s", got)
	}
	if got := NormalizeMetadataMode("PRESERVE"); got != MetadataPreserve {
		t.Fatalf("expected preserve, got %s", got)
	}
	if got := NormalizeMetadataMode("strip"); got != MetadataStrip {
		t.Fatalf("expected strip, got %s", got)
	}
	if got := NormalizeMetadataMode("invalid"); got != "" {
		t.Fatalf("expected empty for invalid mode, got %s", got)
	}
}

func TestMetadataFFmpegArgs(t *testing.T) {
	if got := MetadataFFmpegArgs(MetadataPreserve); got != nil {
		t.Fatalf("expected nil args for preserve, got %#v", got)
	}
	if got := MetadataFFmpegArgs(""); got != nil {
		t.Fatalf("expected nil args for auto, got %#v", got)
	}

	want := []string{"-map_metadata", "-1"}
	got := MetadataFFmpegArgs(MetadataStrip)
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}
