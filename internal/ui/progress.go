package ui

import (
	"fmt"
	"strings"
	"time"
)

// Color ANSI renk kodları
const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
)

// Icons kullanıcı dostu ikonlar
const (
	IconSuccess = "✅"
	IconError   = "❌"
	IconWarning = "⚠️ "
	IconInfo    = "ℹ️ "
	IconConvert = "🔄"
	IconFile    = "📄"
	IconAudio   = "🎵"
	IconImage   = "🖼️ "
	IconVideo   = "🎬"
	IconBatch   = "📦"
	IconDone    = "🎉"
	IconTime    = "⏱️ "
	IconFolder  = "📁"
)

// PrintBanner uygulama başlığını yazdırır
func PrintBanner() {
	banner := `
` + Cyan + Bold + `
  ╔═══════════════════════════════════════════════╗
  ║        FileConverter CLI  v1.2.0              ║
  ║   Yerel dosya format dönüştürücü              ║
  ╚═══════════════════════════════════════════════╝` + Reset + `
`
	fmt.Println(banner)
}

// PrintSuccess başarılı mesaj
func PrintSuccess(msg string) {
	fmt.Printf("%s %s%s%s\n", IconSuccess, Green, msg, Reset)
}

// PrintError hata mesajı
func PrintError(msg string) {
	fmt.Printf("%s %s%s%s\n", IconError, Red, msg, Reset)
}

// PrintWarning uyarı mesajı
func PrintWarning(msg string) {
	fmt.Printf("%s %s%s%s\n", IconWarning, Yellow, msg, Reset)
}

// PrintInfo bilgi mesajı
func PrintInfo(msg string) {
	fmt.Printf("%s %s%s%s\n", IconInfo, Blue, msg, Reset)
}

// PrintConversion dönüştürme işlemi mesajı
func PrintConversion(input, output string) {
	fmt.Printf("%s  %s%s%s → %s%s%s\n", IconConvert, Dim, input, Reset, Green, output, Reset)
}

// PrintDuration süre bilgisi
func PrintDuration(d time.Duration) {
	fmt.Printf("%s  Süre: %s%s%s\n", IconTime, Cyan, formatDuration(d), Reset)
}

// ProgressBar ilerleme çubuğu gösterir
type ProgressBar struct {
	Total   int
	Current int
	Width   int
	Label   string
}

// NewProgressBar yeni bir progress bar oluşturur
func NewProgressBar(total int, label string) *ProgressBar {
	return &ProgressBar{
		Total: total,
		Width: 40,
		Label: label,
	}
}

// Update ilerlemeyi günceller
func (pb *ProgressBar) Update(current int) {
	pb.Current = current
	percentage := float64(current) / float64(pb.Total) * 100
	filled := int(float64(pb.Width) * float64(current) / float64(pb.Total))
	empty := pb.Width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	fmt.Printf("\r  %s%s%s [%s%s%s] %s%.0f%%%s (%d/%d)",
		Bold, pb.Label, Reset,
		Green, bar, Reset,
		Cyan, percentage, Reset,
		current, pb.Total)

	if current >= pb.Total {
		fmt.Println() // Son satırda yeni satıra geç
	}
}

// PrintTable basit bir ASCII tablo yazdırır
func PrintTable(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}

	// Sütun genişliklerini hesapla
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Ayırıcı çizgi
	separator := "  ┼"
	for _, w := range colWidths {
		separator += strings.Repeat("─", w+2) + "┼"
	}

	// Header
	headerLine := "  │"
	for i, h := range headers {
		headerLine += fmt.Sprintf(" %s%-*s%s │", Bold, colWidths[i], h, Reset)
	}

	topLine := "  ┌"
	for _, w := range colWidths {
		topLine += strings.Repeat("─", w+2) + "┬"
	}
	topLine = topLine[:len(topLine)-len("┬")] + "┐"

	bottomLine := "  └"
	for _, w := range colWidths {
		bottomLine += strings.Repeat("─", w+2) + "┴"
	}
	bottomLine = bottomLine[:len(bottomLine)-len("┴")] + "┘"

	separator = "  ├"
	for _, w := range colWidths {
		separator += strings.Repeat("─", w+2) + "┼"
	}
	separator = separator[:len(separator)-len("┼")] + "┤"

	fmt.Println(topLine)
	fmt.Println(headerLine)
	fmt.Println(separator)

	for _, row := range rows {
		line := "  │"
		for i := range headers {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			line += fmt.Sprintf(" %-*s │", colWidths[i], cell)
		}
		fmt.Println(line)
	}

	fmt.Println(bottomLine)
}

// PrintBatchSummary toplu iş özetini yazdırır
func PrintBatchSummary(total, succeeded, skipped, failed int, duration time.Duration) {
	fmt.Println()
	fmt.Printf("  %s %sToplu Dönüşüm Tamamlandı%s\n", IconDone, Bold, Reset)
	fmt.Println("  " + strings.Repeat("─", 40))
	fmt.Printf("  Toplam:    %s%d%s dosya\n", Cyan, total, Reset)
	fmt.Printf("  Başarılı:  %s%d%s dosya\n", Green, succeeded, Reset)
	if skipped > 0 {
		fmt.Printf("  Atlanan:   %s%d%s dosya\n", Yellow, skipped, Reset)
	}
	if failed > 0 {
		fmt.Printf("  Başarısız: %s%d%s dosya\n", Red, failed, Reset)
	}
	fmt.Printf("  Süre:      %s%s%s\n", Yellow, formatDuration(duration), Reset)
	fmt.Println()
}

// formatDuration süreyi okunabilir formata çevirir
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fµs", float64(d.Microseconds()))
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Milliseconds()))
	}
	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}

// PrintFormatCategory format kategorisinin ikonunu döner
func PrintFormatCategory(format string) string {
	documentFormats := map[string]bool{
		"md": true, "html": true, "pdf": true, "docx": true, "txt": true,
		"odt": true, "rtf": true, "csv": true, "xlsx": true,
	}
	audioFormats := map[string]bool{
		"mp3": true, "wav": true, "ogg": true, "flac": true, "aac": true,
		"m4a": true, "wma": true, "opus": true, "webm": true,
	}
	imageFormats := map[string]bool{
		"png": true, "jpg": true, "webp": true, "bmp": true, "gif": true,
		"tif": true, "ico": true,
	}
	videoFormats := map[string]bool{
		"mp4": true, "mov": true, "mkv": true, "avi": true, "webm": true,
		"m4v": true, "wmv": true, "flv": true,
	}

	if documentFormats[format] {
		return IconFile
	}
	if audioFormats[format] {
		return IconAudio
	}
	if imageFormats[format] {
		return IconImage
	}
	if videoFormats[format] {
		return IconVideo
	}
	return IconFile
}
