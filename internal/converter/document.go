package converter

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// DocumentConverter belge dosyalarını dönüştürür
type DocumentConverter struct{}

func init() {
	Register(&DocumentConverter{})
}

func (d *DocumentConverter) Name() string {
	return "Document Converter"
}

func (d *DocumentConverter) SupportedConversions() []ConversionPair {
	return []ConversionPair{
		// Markdown dönüşümleri
		{From: "md", To: "html", Description: "Markdown → HTML"},
		{From: "md", To: "txt", Description: "Markdown → Plain Text"},
		{From: "md", To: "pdf", Description: "Markdown → PDF"},
		{From: "md", To: "docx", Description: "Markdown → DOCX"},
		// HTML dönüşümleri
		{From: "html", To: "txt", Description: "HTML → Plain Text"},
		{From: "html", To: "md", Description: "HTML → Markdown"},
		{From: "html", To: "pdf", Description: "HTML → PDF"},
		{From: "html", To: "docx", Description: "HTML → DOCX"},
		// PDF dönüşümleri
		{From: "pdf", To: "txt", Description: "PDF → Plain Text"},
		{From: "pdf", To: "docx", Description: "PDF → DOCX"},
		{From: "pdf", To: "html", Description: "PDF → HTML"},
		{From: "pdf", To: "md", Description: "PDF → Markdown"},
		// DOCX dönüşümleri
		{From: "docx", To: "txt", Description: "DOCX → Plain Text"},
		{From: "docx", To: "pdf", Description: "DOCX → PDF"},
		{From: "docx", To: "html", Description: "DOCX → HTML"},
		{From: "docx", To: "md", Description: "DOCX → Markdown"},
		// TXT dönüşümleri
		{From: "txt", To: "pdf", Description: "Plain Text → PDF"},
		{From: "txt", To: "html", Description: "Plain Text → HTML"},
		{From: "txt", To: "docx", Description: "Plain Text → DOCX"},
		{From: "txt", To: "md", Description: "Plain Text → Markdown"},
	}
}

func (d *DocumentConverter) SupportsConversion(from, to string) bool {
	for _, pair := range d.SupportedConversions() {
		if pair.From == from && pair.To == to {
			return true
		}
	}
	return false
}

func (d *DocumentConverter) Convert(input string, output string, opts Options) error {
	from := DetectFormat(input)
	to := DetectFormat(output)

	switch {
	// Markdown dönüşümleri
	case from == "md" && to == "html":
		return d.mdToHTML(input, output)
	case from == "md" && to == "txt":
		return d.mdToTxt(input, output)
	case from == "md" && to == "pdf":
		return d.mdToPDF(input, output)
	case from == "md" && to == "docx":
		return d.textToDocx(input, output, true)
	// HTML dönüşümleri
	case from == "html" && to == "txt":
		return d.htmlToTxt(input, output)
	case from == "html" && to == "md":
		return d.htmlToMd(input, output)
	case from == "html" && to == "pdf":
		return d.htmlToPDF(input, output)
	case from == "html" && to == "docx":
		return d.htmlToDocx(input, output)
	// PDF dönüşümleri
	case from == "pdf" && to == "txt":
		return d.pdfToTxt(input, output)
	case from == "pdf" && to == "docx":
		return d.pdfToDocx(input, output)
	case from == "pdf" && to == "html":
		return d.pdfToHTML(input, output)
	case from == "pdf" && to == "md":
		return d.pdfToMd(input, output)
	// DOCX dönüşümleri
	case from == "docx" && to == "txt":
		return d.docxToTxt(input, output)
	case from == "docx" && to == "pdf":
		return d.docxToPDF(input, output)
	case from == "docx" && to == "html":
		return d.docxToHTML(input, output)
	case from == "docx" && to == "md":
		return d.docxToMd(input, output)
	// TXT dönüşümleri
	case from == "txt" && to == "pdf":
		return d.txtToPDF(input, output, opts)
	case from == "txt" && to == "html":
		return d.txtToHTML(input, output)
	case from == "txt" && to == "docx":
		return d.textToDocx(input, output, false)
	case from == "txt" && to == "md":
		return d.txtToMd(input, output)
	default:
		return fmt.Errorf("desteklenmeyen dönüşüm: %s → %s", from, to)
	}
}

// --- Markdown dönüşümleri ---

func (d *DocumentConverter) mdToHTML(input, output string) error {
	source, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("dosya okunamadı: %w", err)
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Table),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithHardWraps(), html.WithXHTML()),
	)

	var buf bytes.Buffer
	buf.WriteString(`<!DOCTYPE html>
<html lang="tr">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Document</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; line-height: 1.6; }
pre { background: #f4f4f4; padding: 16px; border-radius: 8px; overflow-x: auto; }
code { background: #f4f4f4; padding: 2px 6px; border-radius: 4px; }
table { border-collapse: collapse; width: 100%; }
th, td { border: 1px solid #ddd; padding: 8px 12px; text-align: left; }
th { background: #f8f8f8; }
blockquote { border-left: 4px solid #ddd; margin: 0; padding-left: 16px; color: #666; }
</style>
</head>
<body>
`)

	if err := md.Convert(source, &buf); err != nil {
		return fmt.Errorf("markdown dönüşüm hatası: %w", err)
	}

	buf.WriteString("\n</body>\n</html>")

	return os.WriteFile(output, buf.Bytes(), 0644)
}

func (d *DocumentConverter) mdToTxt(input, output string) error {
	source, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("dosya okunamadı: %w", err)
	}

	text := removeMarkdownSyntax(string(source))
	return os.WriteFile(output, []byte(text), 0644)
}

func (d *DocumentConverter) mdToPDF(input, output string) error {
	// Öncelik 1: Pandoc ile pixel-perfect dönüşüm
	if IsPandocAvailable() {
		if err := ConvertWithPandoc(input, output); err == nil {
			return nil
		}
		// Pandoc başarısız olduysa Go renderer'a düş
	}

	// Öncelik 2: LibreOffice ile (MD → HTML → PDF zinciri)
	if IsLibreOfficeAvailable() {
		// Önce HTML'e çevir, sonra LO ile PDF yap
		tmpHTML := output + ".tmp.html"
		if err := d.mdToHTML(input, tmpHTML); err == nil {
			defer os.Remove(tmpHTML)
			if err := ConvertWithLibreOffice(tmpHTML, output, "pdf"); err == nil {
				return nil
			}
		}
	}

	// Öncelik 3: Go renderer (fallback)
	source, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("dosya okunamadı: %w", err)
	}
	return createMarkdownPDF(output, string(source))
}

// ========================================
// PDF Oluşturucu — UTF-8 ve Markdown destekli
// ========================================

// findSystemFont sistemde kullanılabilir bir TTF font yolunu döner
func findSystemFont() string {
	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		candidates = []string{
			"/System/Library/Fonts/Supplemental/Arial.ttf",
			"/Library/Fonts/Arial.ttf",
		}
	case "linux":
		candidates = []string{
			"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
			"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
			"/usr/share/fonts/TTF/DejaVuSans.ttf",
		}
	case "windows":
		candidates = []string{
			"C:\\Windows\\Fonts\\arial.ttf",
			"C:\\Windows\\Fonts\\segoeui.ttf",
		}
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return "" // Bulunamazsa boş döner, fallback kullanılır
}

// initPDFWithFont yeni bir gofpdf oluşturur ve UTF-8 fontları yükler
// hasUTF8 true ise "Sans" ailesi, false ise "Helvetica" ailesi kullanılır
func initPDFWithFont() (*gofpdf.Fpdf, bool) {
	p := gofpdf.New("P", "mm", "A4", "")
	p.SetMargins(20, 20, 20)
	p.SetAutoPageBreak(true, 20)

	fontPath := findSystemFont()
	if fontPath == "" {
		return p, false
	}

	// Font dizinini ve dosya adını ayır
	fontDir := filepath.Dir(fontPath)
	fontFile := filepath.Base(fontPath)
	p.SetFontLocation(fontDir)

	// UTF-8 font ekle — Normal, Bold, Italic
	p.AddUTF8Font("Sans", "", fontFile)
	p.AddUTF8Font("Sans", "B", fontFile)
	p.AddUTF8Font("Sans", "I", fontFile)
	p.AddUTF8Font("Sans", "BI", fontFile)

	return p, true
}

// setFont UTF-8 veya fallback fontu ayarlar
func setFont(p *gofpdf.Fpdf, hasUTF8 bool, style string, size float64) {
	if hasUTF8 {
		p.SetFont("Sans", style, size)
	} else {
		p.SetFont("Helvetica", style, size)
	}
}

// writeText metni yazar, UTF-8 yoksa transliterate eder
func writeText(p *gofpdf.Fpdf, hasUTF8 bool, text string) string {
	if hasUTF8 {
		return text
	}
	return transliterateToLatin(text)
}

// createMarkdownPDF markdown dosyasını biçimlendirmeli PDF'e dönüştürür
func createMarkdownPDF(output string, mdContent string) error {
	p, hasUTF8 := initPDFWithFont()
	p.AddPage()

	lines := strings.Split(mdContent, "\n")
	inCodeBlock := false
	var codeBlockLines []string
	var tableRows [][]string
	inTable := false

	headingSizes := map[int]float64{
		1: 22, 2: 18, 3: 15, 4: 13, 5: 11.5, 6: 11,
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// --- Kod bloğu ---
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if inCodeBlock {
				renderCodeBlockUTF8(p, hasUTF8, codeBlockLines)
				inCodeBlock = false
				codeBlockLines = nil
			} else {
				// Eğer tablo biriktirilmişse önce render et
				if inTable {
					renderTable(p, hasUTF8, tableRows)
					tableRows = nil
					inTable = false
				}
				inCodeBlock = true
				codeBlockLines = nil
			}
			continue
		}

		if inCodeBlock {
			codeBlockLines = append(codeBlockLines, line)
			continue
		}

		trimmed := strings.TrimSpace(line)

		// --- Tablo satırı algılama ---
		if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
			// Ayırıcı satırı atla (|---|---|)
			stripped := strings.ReplaceAll(trimmed, "|", "")
			stripped = strings.ReplaceAll(stripped, "-", "")
			stripped = strings.ReplaceAll(stripped, ":", "")
			stripped = strings.TrimSpace(stripped)
			if stripped == "" {
				continue // ayırıcı satır
			}

			cells := parseTableRow(trimmed)
			tableRows = append(tableRows, cells)
			inTable = true
			continue
		}

		// Tablo bitti, render et
		if inTable {
			renderTable(p, hasUTF8, tableRows)
			tableRows = nil
			inTable = false
		}

		// --- Boş satır ---
		if trimmed == "" {
			p.Ln(3)
			continue
		}

		// --- Yatay çizgi ---
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			y := p.GetY()
			p.SetDrawColor(200, 200, 200)
			p.SetLineWidth(0.4)
			p.Line(20, y+2, 190, y+2)
			p.Ln(6)
			continue
		}

		// --- Başlıklar ---
		headingLevel := 0
		for _, ch := range trimmed {
			if ch == '#' {
				headingLevel++
			} else {
				break
			}
		}
		if headingLevel > 0 && headingLevel <= 6 && len(trimmed) > headingLevel && trimmed[headingLevel] == ' ' {
			headingText := strings.TrimSpace(trimmed[headingLevel+1:])
			headingText = stripInlineMarkdown(headingText)
			fontSize := headingSizes[headingLevel]
			lineHeight := fontSize * 0.5

			if headingLevel <= 2 {
				p.Ln(6)
			} else {
				p.Ln(4)
			}

			safeText := writeText(p, hasUTF8, headingText)
			setFont(p, hasUTF8, "B", fontSize)
			p.MultiCell(0, lineHeight, safeText, "", "", false)

			if headingLevel <= 2 {
				y := p.GetY()
				grayVal := 120
				if headingLevel == 2 {
					grayVal = 180
				}
				p.SetDrawColor(grayVal, grayVal, grayVal)
				p.SetLineWidth(0.3)
				p.Line(20, y+1, 190, y+1)
				p.Ln(3)
			} else {
				p.Ln(2)
			}
			continue
		}

		// --- Blockquote ---
		if strings.HasPrefix(trimmed, "> ") {
			quoteText := strings.TrimPrefix(trimmed, "> ")
			quoteText = stripInlineMarkdown(quoteText)
			safeText := writeText(p, hasUTF8, quoteText)

			y := p.GetY()
			p.SetDrawColor(120, 120, 200)
			p.SetLineWidth(0.8)
			p.Line(22, y, 22, y+7)

			setFont(p, hasUTF8, "I", 10.5)
			p.SetTextColor(100, 100, 100)
			p.SetX(28)
			p.MultiCell(155, 5, safeText, "", "", false)
			p.SetTextColor(0, 0, 0)
			p.Ln(1)
			continue
		}

		// --- Sırasız liste ---
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") {
			listText := trimmed[2:]
			renderFormattedLineUTF8(p, hasUTF8, "   \u2022  "+listText, 10.5)
			continue
		}

		// --- Numaralı liste ---
		if len(trimmed) >= 3 && trimmed[0] >= '0' && trimmed[0] <= '9' {
			dotIdx := strings.Index(trimmed, ". ")
			if dotIdx > 0 && dotIdx <= 3 {
				num := trimmed[:dotIdx]
				listText := trimmed[dotIdx+2:]
				renderFormattedLineUTF8(p, hasUTF8, "   "+num+".  "+listText, 10.5)
				continue
			}
		}

		// --- Normal paragraf ---
		renderFormattedLineUTF8(p, hasUTF8, trimmed, 10.5)
	}

	// Kalan tablo
	if inTable && len(tableRows) > 0 {
		renderTable(p, hasUTF8, tableRows)
	}

	// Kapanmamış kod bloğu
	if inCodeBlock && len(codeBlockLines) > 0 {
		renderCodeBlockUTF8(p, hasUTF8, codeBlockLines)
	}

	return p.OutputFileAndClose(output)
}

// parseTableRow tablo satırını hücrelere böler
func parseTableRow(line string) []string {
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

// renderTable markdown tablosunu PDF'e render eder
func renderTable(p *gofpdf.Fpdf, hasUTF8 bool, rows [][]string) {
	if len(rows) == 0 {
		return
	}

	p.Ln(3)

	// Sütun sayısı
	numCols := len(rows[0])
	if numCols == 0 {
		return
	}

	// Sayfa genişliği
	pageWidth, _ := p.GetPageSize()
	leftMargin, _, rightMargin, _ := p.GetMargins()
	tableWidth := pageWidth - leftMargin - rightMargin
	colWidth := tableWidth / float64(numCols)
	cellHeight := 7.0

	for rowIdx, row := range rows {
		// İlk satır header
		if rowIdx == 0 {
			setFont(p, hasUTF8, "B", 9.5)
			p.SetFillColor(240, 240, 240)
		} else {
			setFont(p, hasUTF8, "", 9.5)
			p.SetFillColor(255, 255, 255)
		}

		p.SetDrawColor(200, 200, 200)

		for j := 0; j < numCols; j++ {
			cellText := ""
			if j < len(row) {
				cellText = stripInlineMarkdown(row[j])
			}
			cellText = writeText(p, hasUTF8, cellText)

			// Metni kırp eğer hücreye sığmıyorsa
			for p.GetStringWidth(cellText) > colWidth-4 && len(cellText) > 3 {
				cellText = cellText[:len(cellText)-4] + "..."
			}

			border := "1"
			p.CellFormat(colWidth, cellHeight, " "+cellText, border, 0, "", rowIdx == 0, 0, "")
		}
		p.Ln(cellHeight)
	}

	p.Ln(3)
}

// renderCodeBlockUTF8 kod bloğunu render eder — sayfa taşmasını parçalayarak çözer
func renderCodeBlockUTF8(p *gofpdf.Fpdf, hasUTF8 bool, codeLines []string) {
	p.Ln(2)

	lineHeight := 4.0
	pageWidth, pageHeight := p.GetPageSize()
	leftMargin, _, rightMargin, bottomMargin := p.GetMargins()
	contentWidth := pageWidth - leftMargin - rightMargin

	// Kod fontu (Courier her zaman Latin-1 ama kod genelde ASCII)
	p.SetFont("Courier", "", 8.5)
	p.SetTextColor(50, 50, 50)

	// Her satırı tek tek render et, sayfa taşması kontrolü ile
	for idx, codeLine := range codeLines {
		// Sayfa kontrolü
		if p.GetY()+lineHeight+4 > pageHeight-bottomMargin {
			p.AddPage()
			p.SetFont("Courier", "", 8.5)
			p.SetTextColor(50, 50, 50)
		}

		// İlk satırda arka plan başlat
		if idx == 0 || p.GetY() <= 25 {
			// Kalan satırlar için blok yüksekliği hesapla
			remainingLines := len(codeLines) - idx
			maxLinesThisPage := int((pageHeight - bottomMargin - p.GetY()) / lineHeight)
			linesThisBlock := remainingLines
			if linesThisBlock > maxLinesThisPage {
				linesThisBlock = maxLinesThisPage
			}
			blockH := float64(linesThisBlock)*lineHeight + 4

			p.SetFillColor(245, 245, 248)
			p.SetDrawColor(220, 220, 225)
			x := p.GetX()
			y := p.GetY()
			p.RoundedRect(x, y, contentWidth, blockH, 1.5, "1234", "FD")
			p.SetY(y)
		}

		p.SetX(leftMargin + 4)
		// Kod metni — transliterate sadece UTF-8 font yoksa
		safeLine := codeLine
		if !hasUTF8 {
			safeLine = transliterateToLatin(codeLine)
		}
		// Uzun satırları kırp
		runeSlice := []rune(safeLine)
		if len(runeSlice) > 90 {
			safeLine = string(runeSlice[:87]) + "..."
		}
		p.CellFormat(contentWidth-8, lineHeight, safeLine, "", 1, "", false, 0, "")
	}

	p.SetTextColor(0, 0, 0)
	p.Ln(4)
}

// renderFormattedLineUTF8 inline markdown formatını (bold, italic, code) destekler
func renderFormattedLineUTF8(p *gofpdf.Fpdf, hasUTF8 bool, line string, baseFontSize float64) {
	lineHeight := baseFontSize * 0.5

	segments := parseInlineMarkdown(line)

	for _, seg := range segments {
		text := writeText(p, hasUTF8, seg.text)
		if text == "" {
			continue
		}

		switch seg.style {
		case "bold":
			setFont(p, hasUTF8, "B", baseFontSize)
		case "italic":
			setFont(p, hasUTF8, "I", baseFontSize)
		case "bolditalic":
			setFont(p, hasUTF8, "BI", baseFontSize)
		case "code":
			p.SetFont("Courier", "", baseFontSize-1)
			p.SetTextColor(180, 50, 50)
		default:
			setFont(p, hasUTF8, "", baseFontSize)
		}

		textWidth := p.GetStringWidth(text)
		pw, _ := p.GetPageSize()
		_, _, rm, _ := p.GetMargins()
		availWidth := pw - p.GetX() - rm

		if textWidth <= availWidth {
			p.CellFormat(textWidth, lineHeight, text, "", 0, "", false, 0, "")
		} else {
			p.MultiCell(0, lineHeight, text, "", "", false)
		}

		if seg.style == "code" {
			p.SetTextColor(0, 0, 0)
		}
	}

	p.Ln(lineHeight)
}

// inlineSegment metin parçasını ve stilini tutar
type inlineSegment struct {
	text  string
	style string // "normal", "bold", "italic", "bolditalic", "code"
}

// parseInlineMarkdown satırdaki **bold**, *italic*, `code` gibi işaretleri ayrıştırır
func parseInlineMarkdown(line string) []inlineSegment {
	var segments []inlineSegment
	remaining := line

	for len(remaining) > 0 {
		// İlk bulunan marker'ı tespit et
		backtickIdx := strings.Index(remaining, "`")
		boldItalicIdx := strings.Index(remaining, "***")
		boldIdx := strings.Index(remaining, "**")
		italicIdx := -1

		// Italic: ilk * bul ama ** olmayan
		for pos := 0; pos < len(remaining); pos++ {
			if remaining[pos] == '*' {
				if pos+1 < len(remaining) && remaining[pos+1] == '*' {
					pos++ // ** atla
					if pos+1 < len(remaining) && remaining[pos+1] == '*' {
						pos++ // *** atla
					}
					continue
				}
				italicIdx = pos
				break
			}
		}

		// En erken marker'ı bul
		type marker struct {
			idx  int
			kind string
		}
		var candidates []marker
		if backtickIdx >= 0 {
			candidates = append(candidates, marker{backtickIdx, "code"})
		}
		if boldItalicIdx >= 0 {
			candidates = append(candidates, marker{boldItalicIdx, "bolditalic"})
		}
		if boldIdx >= 0 && (boldItalicIdx < 0 || boldIdx != boldItalicIdx) {
			candidates = append(candidates, marker{boldIdx, "bold"})
		}
		if italicIdx >= 0 {
			candidates = append(candidates, marker{italicIdx, "italic"})
		}

		if len(candidates) == 0 {
			segments = append(segments, inlineSegment{text: remaining, style: "normal"})
			break
		}

		// En erken olanı seç
		earliest := candidates[0]
		for _, c := range candidates[1:] {
			if c.idx < earliest.idx {
				earliest = c
			}
		}

		// Marker öncesi normal metin
		if earliest.idx > 0 {
			segments = append(segments, inlineSegment{text: remaining[:earliest.idx], style: "normal"})
		}

		var delimLen int
		var delimStr string
		switch earliest.kind {
		case "code":
			delimLen = 1
			delimStr = "`"
		case "bolditalic":
			delimLen = 3
			delimStr = "***"
		case "bold":
			delimLen = 2
			delimStr = "**"
		case "italic":
			delimLen = 1
			delimStr = "*"
		}

		after := remaining[earliest.idx+delimLen:]
		endIdx := strings.Index(after, delimStr)
		if endIdx >= 0 && endIdx > 0 {
			segments = append(segments, inlineSegment{text: after[:endIdx], style: earliest.kind})
			remaining = after[endIdx+delimLen:]
		} else {
			// Kapanmayan marker — normal metin olarak ekle
			segments = append(segments, inlineSegment{text: remaining[earliest.idx : earliest.idx+delimLen], style: "normal"})
			remaining = remaining[earliest.idx+delimLen:]
		}
	}

	return segments
}

// stripInlineMarkdown başlık metinlerinden inline işaretleri temizler
func stripInlineMarkdown(text string) string {
	text = strings.ReplaceAll(text, "***", "")
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "__", "")
	text = strings.ReplaceAll(text, "~~", "")
	text = strings.ReplaceAll(text, "`", "")
	return text
}

// --- HTML dönüşümleri ---

func (d *DocumentConverter) htmlToTxt(input, output string) error {
	source, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("dosya okunamadı: %w", err)
	}

	text := stripHTMLTags(string(source))
	return os.WriteFile(output, []byte(text), 0644)
}

func (d *DocumentConverter) htmlToMd(input, output string) error {
	source, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("dosya okunamadı: %w", err)
	}

	text := string(source)
	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br />", "\n")
	text = strings.ReplaceAll(text, "</p>", "\n\n")
	text = strings.ReplaceAll(text, "<strong>", "**")
	text = strings.ReplaceAll(text, "</strong>", "**")
	text = strings.ReplaceAll(text, "<b>", "**")
	text = strings.ReplaceAll(text, "</b>", "**")
	text = strings.ReplaceAll(text, "<em>", "*")
	text = strings.ReplaceAll(text, "</em>", "*")
	text = strings.ReplaceAll(text, "<i>", "*")
	text = strings.ReplaceAll(text, "</i>", "*")
	text = strings.ReplaceAll(text, "<code>", "`")
	text = strings.ReplaceAll(text, "</code>", "`")

	for i := 6; i >= 1; i-- {
		prefix := strings.Repeat("#", i) + " "
		text = strings.ReplaceAll(text, fmt.Sprintf("<h%d>", i), prefix)
		text = strings.ReplaceAll(text, fmt.Sprintf("</h%d>", i), "\n")
	}

	text = stripHTMLTags(text)
	return os.WriteFile(output, []byte(text), 0644)
}

// --- PDF dönüşümleri ---

func (d *DocumentConverter) pdfToTxt(input, output string) error {
	f, r, err := pdf.Open(input)
	if err != nil {
		return fmt.Errorf("PDF açılamadı: %w", err)
	}
	defer f.Close()

	var buf strings.Builder
	totalPage := r.NumPage()

	for i := 1; i <= totalPage; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		buf.WriteString(text)
		if i < totalPage {
			buf.WriteString(fmt.Sprintf("\n\n--- Sayfa %d ---\n\n", i))
		}
	}

	return os.WriteFile(output, []byte(buf.String()), 0644)
}

// --- DOCX dönüşümleri ---

func (d *DocumentConverter) docxToTxt(input, output string) error {
	r, err := docx.ReadDocxFile(input)
	if err != nil {
		return fmt.Errorf("DOCX açılamadı: %w", err)
	}
	defer r.Close()

	doc := r.Editable()
	content := doc.GetContent()
	content = stripHTMLTags(content)

	return os.WriteFile(output, []byte(content), 0644)
}

// docxToPDF DOCX → PDF
func (d *DocumentConverter) docxToPDF(input, output string) error {
	// Öncelik 1: LibreOffice ile birebir dönüşüm (görseller, tablolar, fontlar korunur)
	if IsLibreOfficeAvailable() {
		if err := ConvertWithLibreOffice(input, output, "pdf"); err == nil {
			return nil
		}
	}

	// Öncelik 2: Metin tabanlı fallback (sadece metin korunur)
	text, err := d.extractDocxText(input)
	if err != nil {
		return err
	}
	return createPDF(output, text, 12)
}

// docxToHTML DOCX → HTML (metin çıkar, HTML template)
func (d *DocumentConverter) docxToHTML(input, output string) error {
	text, err := d.extractDocxText(input)
	if err != nil {
		return err
	}
	return d.textToHTMLFile(output, text)
}

// docxToMd DOCX → Markdown (metin çıkar, basit MD)
func (d *DocumentConverter) docxToMd(input, output string) error {
	text, err := d.extractDocxText(input)
	if err != nil {
		return err
	}
	return os.WriteFile(output, []byte(text), 0644)
}

// extractDocxText DOCX dosyasından düz metin çıkarır
func (d *DocumentConverter) extractDocxText(input string) (string, error) {
	r, err := docx.ReadDocxFile(input)
	if err != nil {
		return "", fmt.Errorf("DOCX açılamadı: %w", err)
	}
	defer r.Close()

	doc := r.Editable()
	content := doc.GetContent()
	return stripHTMLTags(content), nil
}

// --- PDF çapraz dönüşümleri ---

// pdfToDocx PDF → DOCX (metin çıkar, DOCX oluştur)
func (d *DocumentConverter) pdfToDocx(input, output string) error {
	text, err := d.extractPdfText(input)
	if err != nil {
		return err
	}
	return createSimpleDocx(output, text)
}

// pdfToHTML PDF → HTML (metin çıkar, HTML template)
func (d *DocumentConverter) pdfToHTML(input, output string) error {
	text, err := d.extractPdfText(input)
	if err != nil {
		return err
	}
	return d.textToHTMLFile(output, text)
}

// pdfToMd PDF → Markdown (metin çıkar)
func (d *DocumentConverter) pdfToMd(input, output string) error {
	text, err := d.extractPdfText(input)
	if err != nil {
		return err
	}
	return os.WriteFile(output, []byte(text), 0644)
}

// extractPdfText PDF dosyasından düz metin çıkarır
func (d *DocumentConverter) extractPdfText(input string) (string, error) {
	f, r, err := pdf.Open(input)
	if err != nil {
		return "", fmt.Errorf("PDF açılamadı: %w", err)
	}
	defer f.Close()

	var buf strings.Builder
	totalPage := r.NumPage()

	for i := 1; i <= totalPage; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		buf.WriteString(text)
		if i < totalPage {
			buf.WriteString("\n\n")
		}
	}

	return buf.String(), nil
}

// --- HTML çapraz dönüşümleri ---

// htmlToPDF HTML → PDF (metin çıkar, PDF oluştur)
func (d *DocumentConverter) htmlToPDF(input, output string) error {
	// Öncelik 1: LibreOffice ile birebir dönüşüm
	if IsLibreOfficeAvailable() {
		if err := ConvertWithLibreOffice(input, output, "pdf"); err == nil {
			return nil
		}
	}

	// Öncelik 2: Basit metin tabanlı dönüşüm (fallback)
	source, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("dosya okunamadı: %w", err)
	}
	text := stripHTMLTags(string(source))
	return createPDF(output, text, 12)
}

// htmlToDocx HTML → DOCX (metin çıkar, DOCX oluştur)
func (d *DocumentConverter) htmlToDocx(input, output string) error {
	source, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("dosya okunamadı: %w", err)
	}
	text := stripHTMLTags(string(source))
	return createSimpleDocx(output, text)
}

// textToHTMLFile metni styled HTML dosyasına yazar
func (d *DocumentConverter) textToHTMLFile(output, text string) error {
	var buf bytes.Buffer
	buf.WriteString(`<!DOCTYPE html>
<html lang="tr">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Document</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; line-height: 1.6; }
p { margin: 0.5em 0; }
</style>
</head>
<body>
`)

	// Her paragrafı <p> tag'ına sar
	paragraphs := strings.Split(text, "\n\n")
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		para = strings.ReplaceAll(para, "&", "&amp;")
		para = strings.ReplaceAll(para, "<", "&lt;")
		para = strings.ReplaceAll(para, ">", "&gt;")
		para = strings.ReplaceAll(para, "\n", "<br>\n")
		buf.WriteString("<p>" + para + "</p>\n")
	}

	buf.WriteString("</body>\n</html>")
	return os.WriteFile(output, buf.Bytes(), 0644)
}

// --- TXT dönüşümleri ---

// txtToMd TXT → Markdown (doğrudan kopyalama)
func (d *DocumentConverter) txtToMd(input, output string) error {
	source, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("dosya okunamadı: %w", err)
	}
	return os.WriteFile(output, source, 0644)
}

func (d *DocumentConverter) txtToPDF(input, output string, opts Options) error {
	source, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("dosya okunamadı: %w", err)
	}

	fontSize := float64(12)
	if opts.Quality > 0 && opts.Quality <= 100 {
		fontSize = 8 + (float64(opts.Quality)/100.0)*8
	}

	return createPDF(output, string(source), fontSize)
}

func (d *DocumentConverter) txtToHTML(input, output string) error {
	source, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("dosya okunamadı: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(`<!DOCTYPE html>
<html lang="tr">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Document</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; line-height: 1.6; white-space: pre-wrap; }
</style>
</head>
<body>
<pre>`)

	text := strings.ReplaceAll(string(source), "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	buf.WriteString(text)

	buf.WriteString("</pre>\n</body>\n</html>")
	return os.WriteFile(output, buf.Bytes(), 0644)
}

func (d *DocumentConverter) textToDocx(input, output string, isMarkdown bool) error {
	source, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("dosya okunamadı: %w", err)
	}

	content := string(source)
	if isMarkdown {
		content = removeMarkdownSyntax(content)
	}

	return createSimpleDocx(output, content)
}

// ====================================
// Yardımcı fonksiyonlar
// ====================================

func createPDF(output string, content string, fontSize float64) error {
	p, hasUTF8 := initPDFWithFont()
	p.AddPage()
	setFont(p, hasUTF8, "", fontSize)

	lineHeight := fontSize * 0.5
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			p.Ln(lineHeight)
			continue
		}
		safeLine := writeText(p, hasUTF8, line)
		p.MultiCell(0, lineHeight, safeLine, "", "", false)
	}

	return p.OutputFileAndClose(output)
}

func removeMarkdownSyntax(text string) string {
	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		for strings.HasPrefix(trimmed, "#") {
			trimmed = strings.TrimPrefix(trimmed, "#")
		}
		trimmed = strings.TrimSpace(trimmed)

		trimmed = strings.ReplaceAll(trimmed, "**", "")
		trimmed = strings.ReplaceAll(trimmed, "__", "")
		trimmed = strings.ReplaceAll(trimmed, "~~", "")
		trimmed = strings.ReplaceAll(trimmed, "`", "")

		if strings.HasPrefix(trimmed, "- ") {
			trimmed = "• " + strings.TrimPrefix(trimmed, "- ")
		}
		if strings.HasPrefix(trimmed, "* ") {
			trimmed = "• " + strings.TrimPrefix(trimmed, "* ")
		}

		result = append(result, trimmed)
	}

	return strings.Join(result, "\n")
}

func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}

func transliterateToLatin(s string) string {
	replacer := strings.NewReplacer(
		"ç", "c", "Ç", "C",
		"ğ", "g", "Ğ", "G",
		"ı", "i", "İ", "I",
		"ö", "o", "Ö", "O",
		"ş", "s", "Ş", "S",
		"ü", "u", "Ü", "U",
	)
	return replacer.Replace(s)
}

// createSimpleDocx Office Open XML formatında minimal bir DOCX dosyası oluşturur
func createSimpleDocx(outputPath string, content string) error {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`
	if err := addFileToZip(w, "[Content_Types].xml", contentTypes); err != nil {
		return err
	}

	// _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`
	if err := addFileToZip(w, "_rels/.rels", rels); err != nil {
		return err
	}

	// word/_rels/document.xml.rels
	docRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`
	if err := addFileToZip(w, "word/_rels/document.xml.rels", docRels); err != nil {
		return err
	}

	// word/document.xml — asıl içerik
	var docBody strings.Builder
	docBody.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:wpc="http://schemas.microsoft.com/office/word/2010/wordprocessingCanvas"
            xmlns:mo="http://schemas.microsoft.com/office/mac/office/2008/main"
            xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006"
            xmlns:mv="urn:schemas-microsoft-com:mac:vml"
            xmlns:o="urn:schemas-microsoft-com:office:office"
            xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
            xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math"
            xmlns:v="urn:schemas-microsoft-com:vml"
            xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
            xmlns:w10="urn:schemas-microsoft-com:office:word"
            xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
            xmlns:wne="http://schemas.microsoft.com/office/word/2006/wordml"
            xmlns:sl="http://schemas.openxmlformats.org/schemaLibrary/2006/main"
            xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
            xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"
            xmlns:c="http://schemas.openxmlformats.org/drawingml/2006/chart"
            xmlns:lc="http://schemas.openxmlformats.org/drawingml/2006/lockedCanvas"
            xmlns:dgm="http://schemas.openxmlformats.org/drawingml/2006/diagram">
  <w:body>
`)

	// Her satırı bir paragraf olarak ekle
	paragraphs := strings.Split(content, "\n")
	for _, para := range paragraphs {
		// XML escape
		para = strings.ReplaceAll(para, "&", "&amp;")
		para = strings.ReplaceAll(para, "<", "&lt;")
		para = strings.ReplaceAll(para, ">", "&gt;")
		para = strings.ReplaceAll(para, "\"", "&quot;")
		para = strings.ReplaceAll(para, "'", "&apos;")

		docBody.WriteString("    <w:p><w:r><w:t xml:space=\"preserve\">")
		docBody.WriteString(para)
		docBody.WriteString("</w:t></w:r></w:p>\n")
	}

	docBody.WriteString(`  </w:body>
</w:document>`)

	if err := addFileToZip(w, "word/document.xml", docBody.String()); err != nil {
		return err
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("DOCX oluşturulamadı: %w", err)
	}

	return os.WriteFile(outputPath, buf.Bytes(), 0644)
}

func addFileToZip(w *zip.Writer, name string, content string) error {
	f, err := w.Create(name)
	if err != nil {
		return fmt.Errorf("zip dosyası oluşturulamadı (%s): %w", name, err)
	}
	_, err = f.Write([]byte(content))
	if err != nil {
		return fmt.Errorf("zip dosyasına yazılamadı (%s): %w", name, err)
	}
	return nil
}
