package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mlihgenel/fileconverter-cli/internal/installer"
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

var (
	welcomePrimaryColor   = lipgloss.Color("#334155")
	welcomeSecondaryColor = lipgloss.Color("#E2E8F0")
	welcomeTextColor      = lipgloss.Color("#E2E8F0")
	welcomeDimColor       = lipgloss.Color("#94A3B8")
)

// İlk açılış için sade, logo ile uyumlu tonlar
var welcomeGradient = []lipgloss.Color{
	"#F1F5F9", "#E2E8F0", "#CBD5E1", "#94A3B8", "#64748B", "#94A3B8",
}

// Uygulama tanıtım metni
var welcomeDescLines = []string{
	"",
	"  FileConverter'a hos geldiniz!",
	"",
	"  Bu uygulama, dosyalarınızı yerel ortamda güvenli bir şekilde",
	"  dönüştürmenizi sağlar. İnternet'e yükleme gerektirmez.",
	"",
	"  Ozellikler:",
	"",
	"     Belge Donusumu   — MD, HTML, PDF, DOCX, TXT, ODT, RTF, CSV",
	"     Ses Donusumu     — MP3, WAV, OGG, FLAC, AAC, M4A, WMA, OPUS",
	"     Gorsel Donusumu  — PNG, JPEG, WEBP, BMP, GIF, TIFF, ICO",
	"     Video Donusumu   — MP4, MOV, MKV, AVI, WEBM, M4V, WMV, FLV, GIF",
	"",
	"  Toplu donusum ile bir dizindeki tum dosyalari ayni anda",
	"     dönüştürebilirsiniz.",
	"",
	"  Tum islemler tamamen yerel — verileriniz sizde kalir.",
	"",
}

// Feature box stili
var featureBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(welcomePrimaryColor).
	Padding(1, 3).
	MarginLeft(2).
	Width(65)

// ========================================
// Karşılama Ekranı Render
// ========================================

// viewWelcomeIntro animasyonlu karşılama ekranı
func (m interactiveModel) viewWelcomeIntro() string {
	var b strings.Builder

	welcomeSkipStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(welcomeSecondaryColor).
		PaddingLeft(2)

	// Logo renklerinde ASCII banner
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
	versionLine := fmt.Sprintf("v%s  •  Yerel & Güvenli Dönüştürücü", appVersion)
	b.WriteString(lipgloss.NewStyle().Foreground(welcomeDimColor).Italic(true).Render(versionLine))
	b.WriteString("\n")

	// Typing animasyonu — metni charIdx'e kadar göster
	totalChars := 0
	for _, line := range welcomeDescLines {
		lineRunes := []rune(line)
		if totalChars+len(lineRunes) <= m.welcomeCharIdx {
			// Tam satır göster
			b.WriteString(lipgloss.NewStyle().Foreground(welcomeTextColor).Render(line))
			b.WriteString("\n")
			totalChars += len(lineRunes)
		} else {
			// Kısmen göster
			remaining := m.welcomeCharIdx - totalChars
			if remaining > 0 {
				partial := string(lineRunes[:remaining])
				b.WriteString(lipgloss.NewStyle().Foreground(welcomeTextColor).Render(partial))
				// Yanıp sönen cursor
				if m.showCursor {
					b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(welcomeSecondaryColor).Render("▌"))
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

	b.WriteString("\n")
	quickSkipText := "  ⏩ Yazıyı hızlı geçmek için Enter'a basın"
	if m.welcomeCharIdx < totalDesiredChars {
		if m.showCursor {
			b.WriteString(welcomeSkipStyle.Render(quickSkipText))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(welcomeDimColor).Render(quickSkipText))
		}
		b.WriteString("\n")
	}

	if m.welcomeCharIdx >= totalDesiredChars {
		b.WriteString("\n")
		// Yanıp sönen devam mesajı
		continueText := "  ▸ Devam etmek için Enter'a basın"
		if m.showCursor {
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(welcomeSecondaryColor).Render(continueText))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(welcomeDimColor).Render(continueText))
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
		Background(welcomePrimaryColor).
		Padding(0, 2).
		MarginBottom(1)

	welcomeSelectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(welcomeSecondaryColor).
		PaddingLeft(2)

	welcomeNormalStyle := lipgloss.NewStyle().
		Foreground(welcomeTextColor).
		PaddingLeft(4)

	welcomeDimStyle := lipgloss.NewStyle().
		Foreground(welcomeDimColor)

	b.WriteString("\n")
	b.WriteString(titleStyle.Render(" Sistem Kontrolu "))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().Foreground(welcomeTextColor).Render(
		"  Bazı dönüşümler için harici araçlar gereklidir.\n  Durumları kontrol ediliyor...\n"))
	b.WriteString("\n")

	// Bağımlılık durumu tablosu
	hasMissing := false
	for _, dep := range m.dependencies {
		var statusIcon, statusText string
		var style lipgloss.Style

		if dep.Available {
			statusIcon = "OK"
			statusText = "Kurulu"
			style = successStyle
		} else {
			statusIcon = "NO"
			statusText = "Kurulu Değil"
			style = errorStyle
			hasMissing = true
		}

		// Araç ismi
		nameStyle := lipgloss.NewStyle().Bold(true).Foreground(welcomeTextColor).Width(15)
		line := fmt.Sprintf("  %s %s %s",
			statusIcon,
			nameStyle.Render(dep.Name),
			style.Render(statusText))

		if dep.Available && dep.Version != "" {
			ver := dep.Version
			if len(ver) > 40 {
				ver = ver[:40] + "…"
			}
			line += welcomeDimStyle.Render(fmt.Sprintf("  (%s)", ver))
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
				"  Eksik araclar algilandi"))
			b.WriteString("\n\n")

			b.WriteString(welcomeDimStyle.Render(fmt.Sprintf("  Paket yöneticisi: %s", pm)))
			b.WriteString("\n\n")

			// Kurulum seçenekleri
			installOptions := []string{"Eksik araçları otomatik kur", "Atla ve devam et"}
			for i, opt := range installOptions {
				if i == m.cursor {
					b.WriteString(welcomeSelectedStyle.Render(fmt.Sprintf("  ▸ %s", opt)))
				} else {
					b.WriteString(welcomeNormalStyle.Render(fmt.Sprintf("    %s", opt)))
				}
				b.WriteString("\n")
			}
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(warningColor).Render(
				"  Paket yoneticisi bulunamadi. Araclari manuel olarak kurmaniz gerekiyor."))
			b.WriteString("\n\n")

			// Manuel kurulum bilgileri
			for _, dep := range m.dependencies {
				if !dep.Available {
					info := installer.GetInstallInfo(dep.Name)
					b.WriteString(welcomeDimStyle.Render(fmt.Sprintf("  • %s: %s", dep.Name, info.ManualURL)))
					b.WriteString("\n")
				}
			}

			b.WriteString("\n")
			b.WriteString(welcomeDimStyle.Render("  Enter ile devam edin"))
			b.WriteString("\n")
		}
	} else {
		// Tüm araçlar kurulu
		b.WriteString(successStyle.Render("  Tum gerekli araclar kurulu. Hazirsiniz."))
		b.WriteString("\n\n")
		b.WriteString(welcomeDimStyle.Render("  Enter ile devam edin"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(welcomeDimStyle.Render("  ↑↓ Gezin  •  Enter Seç"))
	b.WriteString("\n")

	return b.String()
}

// viewWelcomeInstalling kurulum sırasında gösterilen ekran
func (m interactiveModel) viewWelcomeInstalling() string {
	var b strings.Builder

	b.WriteString("\n\n")

	frame := spinnerFrames[m.spinnerIdx]
	spinnerStyle := lipgloss.NewStyle().Bold(true).Foreground(welcomeSecondaryColor)

	b.WriteString(spinnerStyle.Render(fmt.Sprintf("  %s Araçlar kuruluyor", frame)))

	dots := strings.Repeat(".", (m.spinnerTick/3)%4)
	b.WriteString(lipgloss.NewStyle().Foreground(welcomeDimColor).Render(dots))
	b.WriteString("\n\n")

	if m.installingToolName != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(welcomeDimColor).Render(fmt.Sprintf("  Kurulan: %s", m.installingToolName)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(welcomeDimColor).Render("  Lütfen bekleyin, kurulum devam ediyor..."))
	b.WriteString("\n")

	// Kurulum uyarısı
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(warningColor).Italic(true).Render(
		"  ⓘ Linux'ta sudo şifresi istenebilir."))
	b.WriteString("\n")

	return b.String()
}
