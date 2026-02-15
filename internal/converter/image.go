package converter

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"strings"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
)

// ImageConverter görsel dosyalarını dönüştürür
type ImageConverter struct{}

func init() {
	Register(&ImageConverter{})
}

func (ic *ImageConverter) Name() string {
	return "Image Converter"
}

// imageFormats desteklenen görsel formatları
var imageFormats = []string{"png", "jpg", "webp", "bmp", "gif", "tif"}

// imageWriteFormats yazılabilir formatlar (webp decode-only)
var imageWriteFormats = []string{"png", "jpg", "bmp", "gif", "tif"}

func (ic *ImageConverter) SupportedConversions() []ConversionPair {
	var pairs []ConversionPair
	for _, from := range imageFormats {
		for _, to := range imageWriteFormats {
			if from != to {
				fromDisplay := strings.ToUpper(from)
				toDisplay := strings.ToUpper(to)
				if from == "jpg" {
					fromDisplay = "JPEG"
				}
				if to == "jpg" {
					toDisplay = "JPEG"
				}
				if from == "tif" {
					fromDisplay = "TIFF"
				}
				if to == "tif" {
					toDisplay = "TIFF"
				}
				pairs = append(pairs, ConversionPair{
					From:        from,
					To:          to,
					Description: fmt.Sprintf("%s → %s", fromDisplay, toDisplay),
				})
			}
		}
	}
	return pairs
}

func (ic *ImageConverter) SupportsConversion(from, to string) bool {
	fromOk := false
	toOk := false
	for _, f := range imageFormats {
		if f == from {
			fromOk = true
		}
	}
	for _, f := range imageWriteFormats {
		if f == to {
			toOk = true
		}
	}
	return fromOk && toOk && from != to
}

func (ic *ImageConverter) Convert(input string, output string, opts Options) error {
	from := DetectFormat(input)
	to := DetectFormat(output)

	// Görseli oku
	img, err := ic.decodeImage(input, from)
	if err != nil {
		return err
	}

	// Hedef formata encode et
	return ic.encodeImage(output, img, to, opts.Quality)
}

// decodeImage formatına göre görseli decode eder
func (ic *ImageConverter) decodeImage(path string, format string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("dosya açılamadı: %w", err)
	}
	defer f.Close()

	var img image.Image

	switch format {
	case "png":
		img, err = png.Decode(f)
	case "jpg":
		img, err = jpeg.Decode(f)
	case "gif":
		img, err = gif.Decode(f)
	case "bmp":
		img, err = bmp.Decode(f)
	case "tif":
		img, err = tiff.Decode(f)
	case "webp":
		img, err = webp.Decode(f)
	default:
		// Genel decoder dene
		img, _, err = image.Decode(f)
	}

	if err != nil {
		return nil, fmt.Errorf("görsel decode hatası (%s): %w", format, err)
	}
	return img, nil
}

// encodeImage formatına göre görseli encode eder
func (ic *ImageConverter) encodeImage(path string, img image.Image, format string, quality int) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("çıktı dosyası oluşturulamadı: %w", err)
	}
	defer f.Close()

	switch format {
	case "png":
		err = png.Encode(f, img)
	case "jpg":
		q := 85 // varsayılan JPEG kalitesi
		if quality > 0 && quality <= 100 {
			q = quality
		}
		err = jpeg.Encode(f, img, &jpeg.Options{Quality: q})
	case "gif":
		err = gif.Encode(f, img, nil)
	case "bmp":
		err = bmp.Encode(f, img)
	case "tif":
		err = tiff.Encode(f, img, nil)
	default:
		return fmt.Errorf("desteklenmeyen çıktı formatı: %s", format)
	}

	if err != nil {
		return fmt.Errorf("görsel encode hatası (%s): %w", format, err)
	}
	return nil
}
