package converter

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ========================================
// Harici Araç Entegrasyonları
// LibreOffice (DOCX/HTML → PDF) ve Pandoc (MD → PDF)
// ========================================

// ExternalTool harici bir aracın durumunu temsil eder
type ExternalTool struct {
	Name      string
	Available bool
	Path      string
	Version   string
}

// findLibreOffice sistemde LibreOffice yolunu bulur
func findLibreOffice() (string, error) {
	// 1. Çevre değişkeninden oku
	if envPath := os.Getenv("LIBREOFFICE_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
	}

	// 2. İşletim sistemine göre bilinen yollar
	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		candidates = []string{
			"/Applications/LibreOffice.app/Contents/MacOS/soffice",
			"/usr/local/bin/soffice",
			"/opt/homebrew/bin/soffice",
		}
	case "linux":
		candidates = []string{
			"/usr/bin/soffice",
			"/usr/bin/libreoffice",
			"/usr/local/bin/soffice",
			"/snap/bin/libreoffice",
		}
	case "windows":
		candidates = []string{
			`C:\Program Files\LibreOffice\program\soffice.exe`,
			`C:\Program Files (x86)\LibreOffice\program\soffice.exe`,
		}
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// 3. PATH'te ara
	if path, err := exec.LookPath("soffice"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("libreoffice"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("LibreOffice bulunamadı. Lütfen yükleyin:\n" +
		"  macOS:   brew install --cask libreoffice\n" +
		"  Linux:   sudo apt install libreoffice\n" +
		"  Windows: https://www.libreoffice.org/download\n" +
		"  Veya LIBREOFFICE_PATH çevre değişkenini ayarlayın")
}

// findPandoc sistemde Pandoc yolunu bulur
func findPandoc() (string, error) {
	// 1. Çevre değişkeninden oku
	if envPath := os.Getenv("PANDOC_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
	}

	// 2. PATH'te ara
	if path, err := exec.LookPath("pandoc"); err == nil {
		return path, nil
	}

	// 3. Bilinen yollar
	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		candidates = []string{
			"/usr/local/bin/pandoc",
			"/opt/homebrew/bin/pandoc",
		}
	case "linux":
		candidates = []string{
			"/usr/bin/pandoc",
			"/usr/local/bin/pandoc",
		}
	case "windows":
		candidates = []string{
			`C:\Program Files\Pandoc\pandoc.exe`,
			`C:\Users\` + os.Getenv("USERNAME") + `\AppData\Local\Pandoc\pandoc.exe`,
		}
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("Pandoc bulunamadı. Lütfen yükleyin:\n" +
		"  macOS:   brew install pandoc\n" +
		"  Linux:   sudo apt install pandoc\n" +
		"  Windows: https://pandoc.org/installing.html\n" +
		"  Veya PANDOC_PATH çevre değişkenini ayarlayın")
}

// IsLibreOfficeAvailable LibreOffice'in yüklü olup olmadığını kontrol eder
func IsLibreOfficeAvailable() bool {
	_, err := findLibreOffice()
	return err == nil
}

// IsPandocAvailable Pandoc'un yüklü olup olmadığını kontrol eder
func IsPandocAvailable() bool {
	_, err := findPandoc()
	return err == nil
}

// CheckDependencies tüm harici bağımlılıkları kontrol eder
func CheckDependencies() []ExternalTool {
	tools := []ExternalTool{}

	// LibreOffice
	loTool := ExternalTool{Name: "LibreOffice"}
	if loPath, err := findLibreOffice(); err == nil {
		loTool.Available = true
		loTool.Path = loPath
		// Versiyon al
		if out, err := exec.Command(loPath, "--version").Output(); err == nil {
			loTool.Version = strings.TrimSpace(string(out))
		}
	}
	tools = append(tools, loTool)

	// Pandoc
	pandocTool := ExternalTool{Name: "Pandoc"}
	if pandocPath, err := findPandoc(); err == nil {
		pandocTool.Available = true
		pandocTool.Path = pandocPath
		if out, err := exec.Command(pandocPath, "--version").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 0 {
				pandocTool.Version = strings.TrimSpace(lines[0])
			}
		}
	}
	tools = append(tools, pandocTool)

	// FFmpeg
	ffmpegTool := ExternalTool{Name: "FFmpeg"}
	if IsFFmpegAvailable() {
		ffmpegTool.Available = true
		if path, err := exec.LookPath("ffmpeg"); err == nil {
			ffmpegTool.Path = path
		}
	}
	tools = append(tools, ffmpegTool)

	return tools
}

// ========================================
// LibreOffice Wrapper — DOCX/HTML → PDF
// ========================================

// ConvertWithLibreOffice harici LibreOffice ile dosya dönüştürür
// Desteklenen dönüşümler: docx→pdf, html→pdf, odt→pdf, pptx→pdf vb.
func ConvertWithLibreOffice(inputPath, outputPath, targetFormat string) error {
	soffice, err := findLibreOffice()
	if err != nil {
		return err
	}

	// Girdi dosyasının var olduğunu kontrol et
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("girdi dosyası bulunamadı: %s", inputPath)
	}

	// Çıktı dizinini belirle
	outDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("çıktı dizini oluşturulamadı: %w", err)
	}

	// LibreOffice geçici bir profile ile çalıştır (çakışma önleme)
	tmpProfile, err := os.MkdirTemp("", "libreoffice-profile-*")
	if err != nil {
		return fmt.Errorf("geçici profil oluşturulamadı: %w", err)
	}
	defer os.RemoveAll(tmpProfile)

	// LibreOffice headless komutu
	args := []string{
		"--headless",
		"--norestore",
		"--nologo",
		"-env:UserInstallation=file://" + tmpProfile,
		"--convert-to", targetFormat,
		"--outdir", outDir,
		inputPath,
	}

	cmd := exec.Command(soffice, args...)
	cmd.Stderr = nil
	cmd.Stdout = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("LibreOffice dönüşüm hatası: %w\n"+
			"  Komut: %s %s", err, soffice, strings.Join(args, " "))
	}

	// LibreOffice çıktı dosyasını kontrol et
	// LO, dosyayı girdi dosyasının adıyla çıktı dizinine kaydeder
	inputBase := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	loOutput := filepath.Join(outDir, inputBase+"."+targetFormat)

	// Eğer çıktı dosya adı farklıysa, yeniden adlandır
	if loOutput != outputPath {
		if _, err := os.Stat(loOutput); err == nil {
			if err := os.Rename(loOutput, outputPath); err != nil {
				return fmt.Errorf("çıktı dosyası taşınamadı: %w", err)
			}
		}
	}

	// Çıktı dosyasının oluştuğunu doğrula
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("LibreOffice dönüşüm tamamlandı ancak çıktı dosyası bulunamadı: %s", outputPath)
	}

	return nil
}

// ========================================
// Pandoc Wrapper — Markdown → PDF
// ========================================

// ConvertWithPandoc harici Pandoc ile Markdown dosyayı dönüştürür
func ConvertWithPandoc(inputPath, outputPath string) error {
	pandoc, err := findPandoc()
	if err != nil {
		return err
	}

	// Girdi dosyasının var olduğunu kontrol et
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("girdi dosyası bulunamadı: %s", inputPath)
	}

	// Çıktı dizinini oluştur
	outDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("çıktı dizini oluşturulamadı: %w", err)
	}

	// PDF motoru seç: xelatex > pdflatex > wkhtmltopdf
	pdfEngine := detectPDFEngine()

	var args []string

	if pdfEngine != "" {
		// LaTeX veya wkhtmltopdf tabanlı
		args = []string{
			inputPath,
			"-o", outputPath,
			"--standalone",
			"--pdf-engine=" + pdfEngine,
		}

		// XeLaTeX ise Türkçe karakter desteği için ek ayarlar
		if pdfEngine == "xelatex" || pdfEngine == "lualatex" {
			args = append(args,
				"-V", "mainfont=Arial",
				"-V", "geometry:margin=2.5cm",
				"-V", "lang=tr",
			)
		}
	} else {
		// Motor yoksa HTML ara çıktı ile dene
		args = []string{
			inputPath,
			"-o", outputPath,
			"--standalone",
			"--embed-resources",
			"--self-contained",
		}
	}

	cmd := exec.Command(pandoc, args...)

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return fmt.Errorf("Pandoc dönüşüm hatası: %s", strings.TrimSpace(errMsg))
		}
		return fmt.Errorf("Pandoc dönüşüm hatası: %w", err)
	}

	// Çıktı dosyasının oluştuğunu doğrula
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("Pandoc dönüşüm tamamlandı ancak çıktı dosyası bulunamadı: %s", outputPath)
	}

	return nil
}

// detectPDFEngine kullanılabilir PDF motorunu tespit eder
func detectPDFEngine() string {
	engines := []string{"xelatex", "lualatex", "pdflatex", "wkhtmltopdf"}
	for _, engine := range engines {
		if _, err := exec.LookPath(engine); err == nil {
			return engine
		}
	}
	return ""
}
