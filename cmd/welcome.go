package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/melihgenel/fileconverter-cli/internal/installer"
)

// ========================================
// Karşılama Ekranı — İlk Kullanım
// ========================================

// Hoşgeldin ASCII art
var welcomeArt = []string{
	"",
	"    ███████╗██╗██╗     ███████╗",
	"    ██╔════╝██║██║     ██╔════╝",
	"    █████╗  ██║██║     █████╗  ",
	"    ██╔══╝  ██║██║     ██╔══╝  ",
	"    ██║     ██║███████╗███████╗",
	"    ╚═╝     ╚═╝╚══════╝╚══════╝",
	"",
	"   ██████╗ ██████╗ ███╗   ██╗██╗   ██╗███████╗██████╗ ████████╗███████╗██████╗ ",
	"  ██╔════╝██╔═══██╗████╗  ██║██║   ██║██╔════╝██╔══██╗╚══██╔══╝██╔════╝██╔══██╗",
	"  ██║     ██║   ██║██╔██╗ ██║██║   ██║█████╗  ██████╔╝   ██║   █████╗  ██████╔╝",
	"  ██║     ██║   ██║██║╚██╗██║╚██╗ ██╔╝██╔══╝  ██╔══██╗   ██║   ██╔══╝  ██╔══██╗",
	"  ╚██████╗╚██████╔╝██║ ╚████║ ╚████╔╝ ███████╗██║  ██║   ██║   ███████╗██║  ██║",
	"   ╚═════╝ ╚═════╝ ╚═╝  ╚═══╝  ╚═══╝  ╚══════╝╚═╝  ╚═╝   ╚═╝   ╚══════╝╚═╝  ╚═╝",
	"",
}

// Geniş gradient renkleri (karşılama için)
var welcomeGradient = []lipgloss.Color{
	"#667EEA", "#764BA2", "#F093FB", "#F5576C", "#4FACFE",
	"#00F2FE", "#43E97B", "#FA709A", "#FEE140", "#A18CD1",
}

// Uygulama tanıtım metni
var welcomeDescLines = []string{
	"",
	"  ✨ FileConverter'a hoş geldiniz!",
	"",
	"  Bu uygulama, dosyalarınızı yerel ortamda güvenli bir şekilde",
	"  dönüştürmenizi sağlar. İnternet'e yükleme gerektirmez.",
	"",
	"  🚀 Özellikler:",
	"",
	"     📄  Belge Dönüşümü   — MD, HTML, PDF, DOCX, TXT, ODT, RTF, CSV",
	"     🎵  Ses Dönüşümü     — MP3, WAV, OGG, FLAC, AAC, M4A, WMA, OPUS",
	"     🖼️   Görsel Dönüşümü  — PNG, JPEG, WEBP, BMP, GIF, TIFF, ICO",
	"",
	"  ⚡ Toplu dönüşüm ile bir dizindeki tüm dosyaları aynı anda",
	"     dönüştürebilirsiniz.",
	"",
	"  🔒 Tüm işlemler tamamen yerel — verileriniz sizde kalır.",
	"",
}

// Feature box stili
var featureBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#764BA2")).
	Padding(1, 3).
	MarginLeft(2).
	Width(65)

// ========================================
// Karşılama Ekranı Render
// ========================================

// viewWelcomeIntro animasyonlu karşılama ekranı
func (m interactiveModel) viewWelcomeIntro() string {
	var b strings.Builder

	// Gradient ASCII Art Banner
	for i, line := range welcomeArt {
		if i >= len(welcomeArt) {
			break
		}
		colorIdx := i % len(welcomeGradient)
		style := lipgloss.NewStyle().Bold(true).Foreground(welcomeGradient[colorIdx])
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	// Versiyon bilgisi
	versionLine := fmt.Sprintf("             v%s  •  Yerel & Güvenli Dönüştürücü", appVersion)
	b.WriteString(lipgloss.NewStyle().Foreground(dimTextColor).Italic(true).Render(versionLine))
	b.WriteString("\n")

	// Typing animasyonu — metni charIdx'e kadar göster
	totalChars := 0
	for _, line := range welcomeDescLines {
		lineRunes := []rune(line)
		if totalChars+len(lineRunes) <= m.welcomeCharIdx {
			// Tam satır göster
			b.WriteString(lipgloss.NewStyle().Foreground(textColor).Render(line))
			b.WriteString("\n")
			totalChars += len(lineRunes)
		} else {
			// Kısmen göster
			remaining := m.welcomeCharIdx - totalChars
			if remaining > 0 {
				partial := string(lineRunes[:remaining])
				b.WriteString(lipgloss.NewStyle().Foreground(textColor).Render(partial))
				// Yanıp sönen cursor
				if m.showCursor {
					b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("▌"))
				}
			}
			b.WriteString("\n")
			break
		}
	}

	// Tüm metin gösterildiyse devam mesajı
	totalDesiredChars := 0
	for _, line := range welcomeDescLines {
		totalDesiredChars += len([]rune(line))
	}

	if m.welcomeCharIdx >= totalDesiredChars {
		b.WriteString("\n")
		// Yanıp sönen devam mesajı
		continueText := "  ▸ Devam etmek için Enter'a basın"
		if m.showCursor {
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(accentColor).Render(continueText))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(dimTextColor).Render(continueText))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// viewWelcomeDeps bağımlılık kontrol ve kurulum ekranı
func (m interactiveModel) viewWelcomeDeps() string {
	var b strings.Builder

	// Başlık
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#764BA2")).
		Padding(0, 2).
		MarginBottom(1)

	b.WriteString("\n")
	b.WriteString(titleStyle.Render(" 🔧 Sistem Kontrolü "))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().Foreground(textColor).Render(
		"  Bazı dönüşümler için harici araçlar gereklidir.\n  Durumları kontrol ediliyor...\n"))
	b.WriteString("\n")

	// Bağımlılık durumu tablosu
	hasMissing := false
	for _, dep := range m.dependencies {
		var statusIcon, statusText string
		var style lipgloss.Style

		if dep.Available {
			statusIcon = "✅"
			statusText = "Kurulu"
			style = successStyle
		} else {
			statusIcon = "❌"
			statusText = "Kurulu Değil"
			style = errorStyle
			hasMissing = true
		}

		// Araç ismi
		nameStyle := lipgloss.NewStyle().Bold(true).Foreground(textColor).Width(15)
		line := fmt.Sprintf("  %s %s %s",
			statusIcon,
			nameStyle.Render(dep.Name),
			style.Render(statusText))

		if dep.Available && dep.Version != "" {
			ver := dep.Version
			if len(ver) > 40 {
				ver = ver[:40] + "…"
			}
			line += dimStyle.Render(fmt.Sprintf("  (%s)", ver))
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Eksik araçlar varsa kurulum seçenekleri
	if hasMissing {
		pm := installer.DetectPackageManager()

		if pm != "" {
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(warningColor).Render(
				"  ⚠ Eksik araçlar algılandı!"))
			b.WriteString("\n\n")

			b.WriteString(dimStyle.Render(fmt.Sprintf("  Paket yöneticisi: %s", pm)))
			b.WriteString("\n\n")

			// Kurulum seçenekleri
			installOptions := []string{"Eksik araçları otomatik kur", "Atla ve devam et"}
			for i, opt := range installOptions {
				if i == m.cursor {
					b.WriteString(selectedItemStyle.Render(fmt.Sprintf("  ▸ %s", opt)))
				} else {
					b.WriteString(normalItemStyle.Render(fmt.Sprintf("    %s", opt)))
				}
				b.WriteString("\n")
			}
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(warningColor).Render(
				"  ⚠ Paket yöneticisi bulunamadı. Araçları manuel olarak kurmanız gerekiyor."))
			b.WriteString("\n\n")

			// Manuel kurulum bilgileri
			for _, dep := range m.dependencies {
				if !dep.Available {
					info := installer.GetInstallInfo(dep.Name)
					b.WriteString(dimStyle.Render(fmt.Sprintf("  • %s: %s", dep.Name, info.ManualURL)))
					b.WriteString("\n")
				}
			}

			b.WriteString("\n")
			b.WriteString(dimStyle.Render("  Enter ile devam edin"))
			b.WriteString("\n")
		}
	} else {
		// Tüm araçlar kurulu
		b.WriteString(successStyle.Render("  🎉 Tüm gerekli araçlar kurulu! Hazırsınız."))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("  Enter ile devam edin"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç"))
	b.WriteString("\n")

	return b.String()
}

// viewWelcomeInstalling kurulum sırasında gösterilen ekran
func (m interactiveModel) viewWelcomeInstalling() string {
	var b strings.Builder

	b.WriteString("\n\n")

	frame := spinnerFrames[m.spinnerIdx]
	spinnerStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)

	b.WriteString(spinnerStyle.Render(fmt.Sprintf("  %s Araçlar kuruluyor", frame)))

	dots := strings.Repeat(".", (m.spinnerTick/3)%4)
	b.WriteString(dimStyle.Render(dots))
	b.WriteString("\n\n")

	if m.installingToolName != "" {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Kurulan: %s", m.installingToolName)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Lütfen bekleyin, kurulum devam ediyor..."))
	b.WriteString("\n")

	// Kurulum uyarısı
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(warningColor).Italic(true).Render(
		"  ⓘ Linux'ta sudo şifresi istenebilir."))
	b.WriteString("\n")

	return b.String()
}
