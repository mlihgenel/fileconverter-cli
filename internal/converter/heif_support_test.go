package converter

import "testing"

func TestImageConverterSupportsHEIFSources(t *testing.T) {
	ic := &ImageConverter{}

	if !ic.SupportsConversion("heic", "jpg") {
		t.Fatalf("expected heic -> jpg to be supported")
	}
	if !ic.SupportsConversion("heif", "png") {
		t.Fatalf("expected heif -> png to be supported")
	}
	if ic.SupportsConversion("heic", "heic") {
		t.Fatalf("did not expect heic -> heic to be supported")
	}
}

func TestIsHEIFFormat(t *testing.T) {
	if !IsHEIFFormat("heic") {
		t.Fatalf("expected heic to be recognized")
	}
	if !IsHEIFFormat("HEIF") {
		t.Fatalf("expected heif to be recognized case-insensitively")
	}
	if IsHEIFFormat("png") {
		t.Fatalf("did not expect png to be recognized as heif")
	}
}

func TestIsResizableFormatSupportsHEIF(t *testing.T) {
	if !IsResizableFormat("heic") {
		t.Fatalf("expected heic to be resizable")
	}
	if !IsResizableFormat("heif") {
		t.Fatalf("expected heif to be resizable")
	}
}
