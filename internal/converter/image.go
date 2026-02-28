package converter

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"strings"

	"github.com/HugoSmits86/nativewebp"
	"golang.org/x/image/bmp"
	xdraw "golang.org/x/image/draw"
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
var imageFormats = []string{"png", "jpg", "webp", "bmp", "gif", "tif", "ico", "heic", "heif"}

// imageWriteFormats yazılabilir formatlar
var imageWriteFormats = []string{"png", "jpg", "webp", "bmp", "gif", "tif", "ico"}

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
	return fromOk && toOk
}

func (ic *ImageConverter) Convert(input string, output string, opts Options) error {
	from := DetectFormat(input)
	to := DetectFormat(output)

	// Görseli oku
	img, err := ic.decodeImage(input, from)
	if err != nil {
		return err
	}

	if opts.Resize != nil {
		img, err = ic.resizeImage(img, *opts.Resize)
		if err != nil {
			return err
		}
	}

	// Optimize: kaliteyi otomatik düşür
	quality := opts.Quality
	if opts.Optimize && quality <= 0 {
		switch to {
		case "jpg":
			quality = 60
		}
	}

	// TargetSize: binary search ile kalite yakınsama (sadece lossy formatlar)
	if opts.TargetSize > 0 && to == "jpg" {
		return ic.encodeToTargetSize(output, img, to, opts.TargetSize)
	}

	return ic.encodeImage(output, img, to, quality, opts.Optimize)
}

func (ic *ImageConverter) resizeImage(src image.Image, spec ResizeSpec) (image.Image, error) {
	if spec.Width <= 0 || spec.Height <= 0 {
		return nil, fmt.Errorf("geçersiz hedef boyut: %dx%d", spec.Width, spec.Height)
	}

	srcBounds := src.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()
	if srcWidth <= 0 || srcHeight <= 0 {
		return nil, fmt.Errorf("geçersiz kaynak görsel boyutu")
	}

	switch spec.Mode {
	case ResizeModeStretch:
		return scaleImage(src, spec.Width, spec.Height), nil

	case ResizeModeFit:
		w, h := containSize(srcWidth, srcHeight, spec.Width, spec.Height)
		return scaleImage(src, w, h), nil

	case ResizeModeFill:
		w, h := coverSize(srcWidth, srcHeight, spec.Width, spec.Height)
		scaled := scaleImage(src, w, h)
		return cropCenter(scaled, spec.Width, spec.Height), nil

	case ResizeModePad:
		w, h := containSize(srcWidth, srcHeight, spec.Width, spec.Height)
		scaled := scaleImage(src, w, h)

		canvas := image.NewRGBA(image.Rect(0, 0, spec.Width, spec.Height))
		xdraw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.Black}, image.Point{}, xdraw.Src)

		offsetX := (spec.Width - w) / 2
		offsetY := (spec.Height - h) / 2
		dstRect := image.Rect(offsetX, offsetY, offsetX+w, offsetY+h)
		xdraw.Draw(canvas, dstRect, scaled, scaled.Bounds().Min, xdraw.Over)
		return canvas, nil

	default:
		return nil, fmt.Errorf("desteklenmeyen resize modu: %s", spec.Mode)
	}
}

func scaleImage(src image.Image, width int, height int) image.Image {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}

func containSize(srcWidth int, srcHeight int, targetWidth int, targetHeight int) (int, int) {
	scale := math.Min(float64(targetWidth)/float64(srcWidth), float64(targetHeight)/float64(srcHeight))
	w := int(math.Round(float64(srcWidth) * scale))
	h := int(math.Round(float64(srcHeight) * scale))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

func coverSize(srcWidth int, srcHeight int, targetWidth int, targetHeight int) (int, int) {
	scale := math.Max(float64(targetWidth)/float64(srcWidth), float64(targetHeight)/float64(srcHeight))
	w := int(math.Round(float64(srcWidth) * scale))
	h := int(math.Round(float64(srcHeight) * scale))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

func cropCenter(src image.Image, targetWidth int, targetHeight int) image.Image {
	b := src.Bounds()
	srcWidth := b.Dx()
	srcHeight := b.Dy()

	if targetWidth > srcWidth {
		targetWidth = srcWidth
	}
	if targetHeight > srcHeight {
		targetHeight = srcHeight
	}

	startX := b.Min.X + (srcWidth-targetWidth)/2
	startY := b.Min.Y + (srcHeight-targetHeight)/2

	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	xdraw.Draw(dst, dst.Bounds(), src, image.Point{X: startX, Y: startY}, xdraw.Src)
	return dst
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
	case "ico":
		img, err = decodeICO(f)
	case "heic", "heif":
		img, err = decodeHEIFViaFFmpeg(path)
	default:
		// Genel decoder dene
		img, _, err = image.Decode(f)
	}

	if err != nil {
		return nil, fmt.Errorf("görsel decode hatası (%s): %w", format, err)
	}
	return img, nil
}

func decodeHEIFViaFFmpeg(path string) (image.Image, error) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("heic/heif decode için ffmpeg gerekli")
	}

	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-i", path,
		"-frames:v", "1",
		"-f", "image2pipe",
		"-vcodec", "png",
		"-",
	}
	out, err := exec.Command(ffmpegPath, args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return nil, fmt.Errorf("heic/heif ffmpeg decode hatası: %w", err)
		}
		return nil, fmt.Errorf("heic/heif ffmpeg decode hatası: %s", msg)
	}

	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		return nil, fmt.Errorf("heic/heif png decode hatası: %w", err)
	}
	return img, nil
}

// encodeImage formatına göre görseli encode eder
func (ic *ImageConverter) encodeImage(path string, img image.Image, format string, quality int, optimize bool) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("çıktı dosyası oluşturulamadı: %w", err)
	}
	defer f.Close()

	switch format {
	case "png":
		enc := &png.Encoder{}
		if optimize {
			enc.CompressionLevel = png.BestCompression
		}
		err = enc.Encode(f, img)
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
	case "webp":
		err = nativewebp.Encode(f, img, nil)
	case "ico":
		err = encodeICO(f, img)
	default:
		return fmt.Errorf("desteklenmeyen çıktı formatı: %s", format)
	}

	if err != nil {
		return fmt.Errorf("görsel encode hatası (%s): %w", format, err)
	}
	return nil
}

// encodeToTargetSize binary search ile JPEG kalitesini hedef dosya boyutuna yakınsar
func (ic *ImageConverter) encodeToTargetSize(path string, img image.Image, format string, targetSize int64) error {
	minQ, maxQ := 10, 95
	bestQ := 80
	tolerance := 0.15 // ±%15

	for i := 0; i < 8; i++ {
		midQ := (minQ + maxQ) / 2

		// Buffer'a encode et
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: midQ}); err != nil {
			return fmt.Errorf("optimize encode hatası: %w", err)
		}

		size := int64(buf.Len())
		bestQ = midQ

		// Hedef boyuta yeterince yakınsa dur
		ratio := float64(size) / float64(targetSize)
		if ratio >= 1.0-tolerance && ratio <= 1.0+tolerance {
			break
		}

		if size > targetSize {
			maxQ = midQ - 1
		} else {
			minQ = midQ + 1
		}

		if minQ > maxQ {
			break
		}
	}

	// En iyi kalite ile dosyaya yaz
	return ic.encodeImage(path, img, format, bestQ, false)
}

// decodeICO ICO dosyasından ilk görseli okur (PNG veya BMP sub-image)
func decodeICO(r io.ReadSeeker) (image.Image, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// ICO header: 6 byte (reserved=0, type=1, count)
	if len(data) < 6 {
		return nil, fmt.Errorf("geçersiz ICO dosyası")
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count == 0 {
		return nil, fmt.Errorf("ICO dosyasında görsel yok")
	}

	// İlk entry: offset 6, her entry 16 byte
	if len(data) < 6+16 {
		return nil, fmt.Errorf("geçersiz ICO entry")
	}
	offset := int(binary.LittleEndian.Uint32(data[6+12 : 6+16]))
	size := int(binary.LittleEndian.Uint32(data[6+8 : 6+12]))

	if offset+size > len(data) {
		return nil, fmt.Errorf("geçersiz ICO veri aralığı")
	}

	imgData := data[offset : offset+size]

	// PNG mi BMP mi kontrol et
	if len(imgData) >= 8 && imgData[0] == 0x89 && imgData[1] == 'P' {
		return png.Decode(bytes.NewReader(imgData))
	}
	// BMP sub-image (DIB header)
	return bmp.Decode(bytes.NewReader(imgData))
}

// encodeICO görseli minimal ICO formatında yazar (PNG payload)
func encodeICO(w io.Writer, img image.Image) error {
	// Önce PNG'ye encode et
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return err
	}
	pngData := pngBuf.Bytes()

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width > 255 {
		width = 0 // 0 = 256px in ICO spec
	}
	if height > 255 {
		height = 0
	}

	// ICO header (6 bytes)
	header := make([]byte, 6)
	binary.LittleEndian.PutUint16(header[0:2], 0) // Reserved
	binary.LittleEndian.PutUint16(header[2:4], 1) // Type: ICO
	binary.LittleEndian.PutUint16(header[4:6], 1) // Count: 1

	// Directory entry (16 bytes)
	entry := make([]byte, 16)
	entry[0] = byte(width)
	entry[1] = byte(height)
	entry[2] = 0                                                     // Color palette
	entry[3] = 0                                                     // Reserved
	binary.LittleEndian.PutUint16(entry[4:6], 1)                     // Color planes
	binary.LittleEndian.PutUint16(entry[6:8], 32)                    // Bits per pixel
	binary.LittleEndian.PutUint32(entry[8:12], uint32(len(pngData))) // Size
	binary.LittleEndian.PutUint32(entry[12:16], 22)                  // Offset (6 + 16 = 22)

	if _, err := w.Write(header); err != nil {
		return err
	}
	if _, err := w.Write(entry); err != nil {
		return err
	}
	_, err := w.Write(pngData)
	return err
}
