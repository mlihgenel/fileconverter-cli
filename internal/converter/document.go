package converter

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
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
		{From: "md", To: "html", Description: "Markdown → HTML"},
		{From: "md", To: "txt", Description: "Markdown → Plain Text"},
		{From: "md", To: "pdf", Description: "Markdown → PDF"},
		{From: "md", To: "docx", Description: "Markdown → DOCX"},
		{From: "html", To: "txt", Description: "HTML → Plain Text"},
		{From: "html", To: "md", Description: "HTML → Markdown"},
		{From: "pdf", To: "txt", Description: "PDF → Plain Text"},
		{From: "docx", To: "txt", Description: "DOCX → Plain Text"},
		{From: "txt", To: "pdf", Description: "Plain Text → PDF"},
		{From: "txt", To: "html", Description: "Plain Text → HTML"},
		{From: "txt", To: "docx", Description: "Plain Text → DOCX"},
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
	case from == "md" && to == "html":
		return d.mdToHTML(input, output)
	case from == "md" && to == "txt":
		return d.mdToTxt(input, output)
	case from == "md" && to == "pdf":
		return d.mdToPDF(input, output)
	case from == "md" && to == "docx":
		return d.textToDocx(input, output, true)
	case from == "html" && to == "txt":
		return d.htmlToTxt(input, output)
	case from == "html" && to == "md":
		return d.htmlToMd(input, output)
	case from == "pdf" && to == "txt":
		return d.pdfToTxt(input, output)
	case from == "docx" && to == "txt":
		return d.docxToTxt(input, output)
	case from == "txt" && to == "pdf":
		return d.txtToPDF(input, output, opts)
	case from == "txt" && to == "html":
		return d.txtToHTML(input, output)
	case from == "txt" && to == "docx":
		return d.textToDocx(input, output, false)
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
	source, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("dosya okunamadı: %w", err)
	}

	text := removeMarkdownSyntax(string(source))
	return createPDF(output, text, 12)
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

// --- TXT dönüşümleri ---

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
	p := gofpdf.New("P", "mm", "A4", "")
	p.SetMargins(20, 20, 20)
	p.AddPage()
	p.SetFont("Helvetica", "", fontSize)

	lineHeight := fontSize * 0.5
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			p.Ln(lineHeight)
			continue
		}
		safeLine := transliterateToLatin(line)
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
