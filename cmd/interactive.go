package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/melihgenel/fileconverter/internal/converter"
)

// ========================================
// Renk Paleti ve Stiller
// ========================================

var (
	// Ana renk paleti
	primaryColor   = lipgloss.Color("#7C3AED") // Mor
	secondaryColor = lipgloss.Color("#06B6D4") // Cyan
	accentColor    = lipgloss.Color("#10B981") // Yeşil
	warningColor   = lipgloss.Color("#F59E0B") // Sarı
	dangerColor    = lipgloss.Color("#EF4444") // Kırmızı
	textColor      = lipgloss.Color("#E2E8F0") // Açık gri
	dimTextColor   = lipgloss.Color("#64748B") // Koyu gri
	bgColor        = lipgloss.Color("#0F172A") // Koyu arka plan

	// Gradient renkleri (banner için)
	gradientColors = []lipgloss.Color{
		"#818CF8", "#A78BFA", "#C084FC", "#E879F9", "#F472B6",
	}

	// Stiller
	bannerStyle = lipgloss.NewStyle().
			Bold(true).
			MarginBottom(1)

	menuTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Padding(0, 2).
			MarginBottom(1)

	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(secondaryColor).
				PaddingLeft(2)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(textColor).
			PaddingLeft(4)

	descStyle = lipgloss.NewStyle().
			Foreground(dimTextColor).
			Italic(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(dimTextColor)

	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(dangerColor)

	infoStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	pathStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)

	resultBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 3).
			MarginTop(1)

	breadcrumbStyle = lipgloss.NewStyle().
			Foreground(dimTextColor).
			PaddingLeft(2)

	selectedFileStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor).
				PaddingLeft(2)

	folderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(warningColor)

	spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
)

// ========================================
// Kategori tanımları
// ========================================

type formatCategory struct {
	Name    string
	Icon    string
	Desc    string
	Formats []string
}

var categories = []formatCategory{
	{Name: "Belgeler", Icon: "📄", Desc: "MD, HTML, PDF, DOCX, TXT — çapraz dönüşüm", Formats: []string{"md", "html", "pdf", "docx", "txt"}},
	{Name: "Ses Dosyaları", Icon: "🎵", Desc: "MP3, WAV, OGG, FLAC, AAC, M4A, WMA", Formats: []string{"mp3", "wav", "ogg", "flac", "aac", "m4a", "wma"}},
	{Name: "Görseller", Icon: "🖼️ ", Desc: "PNG, JPEG, WEBP, BMP, GIF, TIFF", Formats: []string{"png", "jpg", "webp", "bmp", "gif", "tif"}},
}

// ========================================
// State Machine
// ========================================

type screenState int

const (
	stateMainMenu screenState = iota
	stateSelectCategory
	stateSelectSourceFormat
	stateSelectTargetFormat
	stateFileBrowser
	stateConverting
	stateConvertDone
	stateBatchSelectCategory
	stateBatchSelectSourceFormat
	stateBatchSelectTargetFormat
	stateBatchConverting
	stateBatchDone
	stateFormats
	stateDependencies
)

// ========================================
// Model
// ========================================

type interactiveModel struct {
	state  screenState
	cursor int

	// Menü
	choices     []string
	choiceIcons []string
	choiceDescs []string

	// Kategori
	selectedCategory int

	// Dönüşüm bilgileri
	sourceFormat string
	targetFormat string
	selectedFile string

	// Dosya tarayıcı
	browserDir    string
	browserItems  []browserEntry
	defaultOutput string

	// Sonuçlar
	resultMsg string
	resultErr bool
	duration  time.Duration

	// Batch
	batchTotal     int
	batchSucceeded int
	batchFailed    int

	// Spinner
	spinnerIdx  int
	spinnerTick int

	// Pencere
	width  int
	height int

	// Çıkış
	quitting bool

	// Sistem durumu
	dependencies []converter.ExternalTool
}

type browserEntry struct {
	name  string
	path  string
	isDir bool
}

// Mesajlar
type convertDoneMsg struct {
	err      error
	duration time.Duration
	output   string
}

type batchDoneMsg struct {
	total     int
	succeeded int
	failed    int
	duration  time.Duration
}

type tickMsg time.Time

func newInteractiveModel(deps []converter.ExternalTool) interactiveModel {
	homeDir := getHomeDir()
	desktop := filepath.Join(homeDir, "Desktop")

	return interactiveModel{
		state:  stateMainMenu,
		cursor: 0,
		choices: []string{
			"Dosya Dönüştür",
			"Toplu Dönüştür (Batch)",
			"Desteklenen Formatlar",
			"Sistem Kontrolü",
			"Çıkış",
		},
		choiceIcons: []string{"🔄", "📦", "📋", "🔧", "👋"},
		choiceDescs: []string{
			"Tek bir dosyayı başka formata dönüştür",
			"Dizindeki tüm dosyaları toplu dönüştür",
			"Desteklenen format ve dönüşüm yollarını gör",
			"Harici araçların (LibreOffice, Pandoc) durumu",
			"Uygulamadan çık",
		},
		browserDir:    desktop,
		defaultOutput: desktop,
		width:         80,
		height:        24,
		dependencies:  deps,
	}
}

// ========================================
// bubbletea Interface
// ========================================

func (m interactiveModel) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m interactiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if m.state == stateConverting || m.state == stateBatchConverting {
			m.spinnerTick++
			m.spinnerIdx = m.spinnerTick % len(spinnerFrames)
		}
		return m, tickCmd()

	case convertDoneMsg:
		m.state = stateConvertDone
		if msg.err != nil {
			m.resultMsg = msg.err.Error()
			m.resultErr = true
		} else {
			m.resultMsg = msg.output
			m.resultErr = false
		}
		m.duration = msg.duration
		return m, nil

	case batchDoneMsg:
		m.state = stateBatchDone
		m.batchTotal = msg.total
		m.batchSucceeded = msg.succeeded
		m.batchFailed = msg.failed
		m.duration = msg.duration
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "q":
			if m.state == stateMainMenu {
				m.quitting = true
				return m, tea.Quit
			}
			return m.goToMainMenu(), nil

		case "esc":
			return m.goBack(), nil

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			max := m.getMaxCursor()
			if m.cursor < max {
				m.cursor++
			}

		case "enter":
			return m.handleEnter()
		}
	}

	return m, nil
}

func (m interactiveModel) getMaxCursor() int {
	switch m.state {
	case stateFileBrowser:
		return len(m.browserItems) - 1
	case stateFormats:
		return 0
	default:
		return len(m.choices) - 1
	}
}

func (m interactiveModel) View() string {
	if m.quitting {
		return gradientText("  👋 Görüşürüz!", gradientColors) + "\n\n"
	}

	switch m.state {
	case stateMainMenu:
		return m.viewMainMenu()
	case stateSelectCategory:
		return m.viewSelectCategory("Dosya türü seçin:")
	case stateSelectSourceFormat:
		return m.viewSelectFormat("Kaynak format seçin:")
	case stateSelectTargetFormat:
		return m.viewSelectFormat("Hedef format seçin:")
	case stateFileBrowser:
		return m.viewFileBrowser()
	case stateConverting, stateBatchConverting:
		return m.viewConverting()
	case stateConvertDone:
		return m.viewConvertDone()
	case stateBatchSelectCategory:
		return m.viewSelectCategory("Batch — Dosya türü seçin:")
	case stateBatchSelectSourceFormat:
		return m.viewSelectFormat("Batch — Kaynak format seçin:")
	case stateBatchSelectTargetFormat:
		return m.viewSelectFormat("Batch — Hedef format seçin:")
	case stateBatchDone:
		return m.viewBatchDone()
	case stateFormats:
		return m.viewFormats()
	case stateDependencies:
		return m.viewDependencies()
	default:
		return ""
	}
}

// ========================================
// Ekranlar
// ========================================

func (m interactiveModel) viewMainMenu() string {
	var b strings.Builder

	// Gradient banner
	banner := []string{
		"  ╔════════════════════════════════════════════════════╗",
		"  ║                                                    ║",
		"  ║     		███████╗██╗██╗     ███████╗             ║",
		"  ║     		██╔════╝██║██║     ██╔════╝             ║",
		"  ║     		█████╗  ██║██║     █████╗               ║",
		"  ║     		██╔══╝  ██║██║     ██╔══╝               ║",
		"  ║     		██║     ██║███████╗███████╗             ║",
		"  ║     		╚═╝     ╚═╝╚══════╝╚══════╝             ║",
		"  ║        F I L E   C O N V E R T E R   v1.0.0        ║",
		"  ║                                                    ║",
		"  ║     Dosyalarınızı yerel ve güvenli dönüştürün      ║",
		"  ╚════════════════════════════════════════════════════╝",
	}

	for i, line := range banner {
		colorIdx := i % len(gradientColors)
		style := lipgloss.NewStyle().Bold(true).Foreground(gradientColors[colorIdx])
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" ◆ Ana Menü "))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		icon := m.choiceIcons[i]
		desc := ""
		if i < len(m.choiceDescs) {
			desc = m.choiceDescs[i]
		}

		if i == m.cursor {
			b.WriteString(selectedItemStyle.Render(fmt.Sprintf("▸ %s  %s", icon, choice)))
			b.WriteString("\n")
			if desc != "" {
				b.WriteString(lipgloss.NewStyle().PaddingLeft(7).Foreground(dimTextColor).Italic(true).Render(desc))
				b.WriteString("\n")
			}
		} else {
			b.WriteString(normalItemStyle.Render(fmt.Sprintf("  %s  %s", icon, choice)))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç  •  q Çıkış"))
	b.WriteString("\n")

	return b.String()
}

func (m interactiveModel) viewSelectCategory(title string) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(fmt.Sprintf(" ◆ %s ", title)))
	b.WriteString("\n\n")

	for i, cat := range categories {
		if i == m.cursor {
			// Seçili kategori — kart stili
			card := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(secondaryColor).
				Padding(0, 2).
				MarginLeft(2).
				Width(50)

			content := fmt.Sprintf("%s  %s\n%s",
				cat.Icon,
				lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render(cat.Name),
				descStyle.Render(cat.Desc))

			b.WriteString(card.Render(content))
		} else {
			b.WriteString(normalItemStyle.Render(fmt.Sprintf("  %s  %s", cat.Icon, cat.Name)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç  •  Esc Geri"))
	b.WriteString("\n")

	return b.String()
}

func (m interactiveModel) viewSelectFormat(title string) string {
	var b strings.Builder

	b.WriteString("\n")

	// Breadcrumb
	cat := categories[m.selectedCategory]
	crumb := fmt.Sprintf("  %s %s", cat.Icon, cat.Name)
	if m.sourceFormat != "" {
		crumb += fmt.Sprintf(" › %s", lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render(strings.ToUpper(m.sourceFormat)))
	}
	b.WriteString(breadcrumbStyle.Render(crumb))
	b.WriteString("\n\n")

	b.WriteString(menuTitleStyle.Render(fmt.Sprintf(" ◆ %s ", title)))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		icon := ""
		if i < len(m.choiceIcons) {
			icon = m.choiceIcons[i]
		}
		if i == m.cursor {
			b.WriteString(selectedItemStyle.Render(fmt.Sprintf("▸ %s  %s", icon, choice)))
		} else {
			b.WriteString(normalItemStyle.Render(fmt.Sprintf("  %s  %s", icon, choice)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç  •  Esc Geri"))
	b.WriteString("\n")

	return b.String()
}

func (m interactiveModel) viewFileBrowser() string {
	var b strings.Builder

	b.WriteString("\n")

	// Breadcrumb
	cat := categories[m.selectedCategory]
	crumb := fmt.Sprintf("  %s %s › %s › %s",
		cat.Icon, cat.Name,
		lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render(strings.ToUpper(m.sourceFormat)),
		lipgloss.NewStyle().Bold(true).Foreground(accentColor).Render(strings.ToUpper(m.targetFormat)))
	b.WriteString(breadcrumbStyle.Render(crumb))
	b.WriteString("\n\n")

	b.WriteString(menuTitleStyle.Render(" ◆ Dosya Seçin "))
	b.WriteString("\n")

	// Mevcut dizin
	shortDir := shortenPath(m.browserDir)
	b.WriteString(pathStyle.Render(fmt.Sprintf("  📁 %s", shortDir)))
	b.WriteString("\n\n")

	if len(m.browserItems) == 0 {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Bu dizinde .%s dosyası veya klasör bulunamadı!", m.sourceFormat)))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("  Esc Geri"))
		b.WriteString("\n")
		return b.String()
	}

	// Sayfala
	pageSize := 15
	startIdx := 0
	if m.cursor >= pageSize {
		startIdx = m.cursor - pageSize + 1
	}
	endIdx := startIdx + pageSize
	if endIdx > len(m.browserItems) {
		endIdx = len(m.browserItems)
	}

	for i := startIdx; i < endIdx; i++ {
		item := m.browserItems[i]

		if item.isDir {
			// Klasörler
			if i == m.cursor {
				b.WriteString(selectedItemStyle.Render(fmt.Sprintf("▸ 📁 %s/", item.name)))
			} else {
				b.WriteString(normalItemStyle.Render(fmt.Sprintf("  📁 %s/", folderStyle.Render(item.name))))
			}
		} else {
			// Dosyalar
			if i == m.cursor {
				b.WriteString(selectedFileStyle.Render(fmt.Sprintf("▸ %s %s", cat.Icon, item.name)))
			} else {
				b.WriteString(normalItemStyle.Render(fmt.Sprintf("  %s %s", cat.Icon, item.name)))
			}
		}
		b.WriteString("\n")
	}

	// Bilgi
	fileCount := 0
	dirCount := 0
	for _, item := range m.browserItems {
		if item.isDir {
			dirCount++
		} else {
			fileCount++
		}
	}

	b.WriteString("\n")
	info := fmt.Sprintf("  %d dosya", fileCount)
	if dirCount > 0 {
		info += fmt.Sprintf(", %d klasör", dirCount)
	}
	b.WriteString(infoStyle.Render(info))
	if len(m.browserItems) > pageSize {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  (%d-%d arası)", startIdx+1, endIdx)))
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç/Gir  •  Esc Geri"))
	b.WriteString("\n")

	// Çıktı bilgisi
	b.WriteString(dimStyle.Render(fmt.Sprintf("  💾 Çıktı: %s", shortenPath(m.defaultOutput))))
	b.WriteString("\n")

	return b.String()
}

func (m interactiveModel) viewConverting() string {
	var b strings.Builder
	b.WriteString("\n\n")

	frame := spinnerFrames[m.spinnerIdx]
	spinnerStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)

	b.WriteString(spinnerStyle.Render(fmt.Sprintf("  %s Dönüştürülüyor", frame)))

	dots := strings.Repeat(".", (m.spinnerTick/3)%4)
	b.WriteString(dimStyle.Render(dots))
	b.WriteString("\n\n")

	if m.selectedFile != "" {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %s → %s",
			filepath.Base(m.selectedFile),
			strings.ToUpper(m.targetFormat))))
		b.WriteString("\n")
	}

	return b.String()
}

func (m interactiveModel) viewConvertDone() string {
	var b strings.Builder

	b.WriteString("\n")
	if m.resultErr {
		content := errorStyle.Render("  ❌ Dönüşüm Başarısız") + "\n\n"
		content += fmt.Sprintf("  Hata: %s", m.resultMsg)
		b.WriteString(resultBoxStyle.Render(content))
	} else {
		content := successStyle.Render("  ✅ Dönüşüm Tamamlandı!") + "\n\n"
		content += fmt.Sprintf("  📄 Çıktı: %s\n", shortenPath(m.resultMsg))
		content += fmt.Sprintf("  ⏱️  Süre:  %s", formatDuration(m.duration))
		b.WriteString(resultBoxStyle.Render(content))
	}

	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("  Enter Ana Menü  •  Esc Geri"))
	b.WriteString("\n")

	return b.String()
}

func (m interactiveModel) viewBatchDone() string {
	var b strings.Builder

	b.WriteString("\n")

	content := successStyle.Render("  🎉 Toplu Dönüşüm Tamamlandı!") + "\n\n"
	content += fmt.Sprintf("  Toplam:    %d dosya\n", m.batchTotal)
	content += successStyle.Render(fmt.Sprintf("  Başarılı:  %d dosya\n", m.batchSucceeded))
	if m.batchFailed > 0 {
		content += errorStyle.Render(fmt.Sprintf("  Başarısız: %d dosya\n", m.batchFailed))
	}
	content += fmt.Sprintf("  Süre:      %s", formatDuration(m.duration))

	b.WriteString(resultBoxStyle.Render(content))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("  Enter Ana Menü"))
	b.WriteString("\n")

	return b.String()
}

func (m interactiveModel) viewFormats() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" ◆ Desteklenen Dönüşümler "))
	b.WriteString("\n\n")

	pairs := converter.GetAllConversions()

	docFormats := map[string]bool{"md": true, "html": true, "pdf": true, "docx": true, "txt": true}
	audioFormats := map[string]bool{"mp3": true, "wav": true, "ogg": true, "flac": true, "aac": true, "m4a": true, "wma": true}
	imgFormats := map[string]bool{"png": true, "jpg": true, "webp": true, "bmp": true, "gif": true, "tif": true}

	// Belge formatları
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("  📄 Belge Formatları"))
	b.WriteString("\n")
	for _, p := range pairs {
		if docFormats[p.From] && docFormats[p.To] {
			b.WriteString(fmt.Sprintf("     %s → %s\n",
				lipgloss.NewStyle().Bold(true).Foreground(textColor).Render(strings.ToUpper(p.From)),
				successStyle.Render(strings.ToUpper(p.To))))
		}
	}

	// Ses
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("  🎵 Ses Formatları"))
	if !converter.IsFFmpegAvailable() {
		b.WriteString(errorStyle.Render("  ⚠ FFmpeg gerekli"))
	}
	b.WriteString("\n")
	audioList := sortedKeys(audioFormats)
	b.WriteString(fmt.Sprintf("     %s\n", dimStyle.Render(strings.Join(audioList, " ↔ ")+"  (çapraz)")))

	// Görsel
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("  🖼️  Görsel Formatları"))
	b.WriteString("\n")
	imgList := sortedKeys(imgFormats)
	b.WriteString(fmt.Sprintf("     %s\n", dimStyle.Render(strings.Join(imgList, " ↔ ")+"  (çapraz)")))

	b.WriteString("\n")
	b.WriteString(infoStyle.Render(fmt.Sprintf("  Toplam: %d dönüşüm yolu", len(pairs))))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("  Esc Ana Menü"))
	b.WriteString("\n")

	return b.String()
}

// ========================================
// İşlem Mantığı
// ========================================

func (m interactiveModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.state {
	case stateMainMenu:
		switch m.cursor {
		case 0:
			return m.goToCategorySelect(false), nil
		case 1:
			return m.goToCategorySelect(true), nil
		case 2:
			m.state = stateFormats
			m.cursor = 0
			return m, nil
		case 3:
			m.state = stateDependencies
			m.cursor = 0
			return m, nil
		case 4:
			m.quitting = true
			return m, tea.Quit
		}

	case stateSelectCategory:
		m.selectedCategory = m.cursor
		return m.goToSourceFormatSelect(false), nil

	case stateSelectSourceFormat:
		m.sourceFormat = converter.NormalizeFormat(m.choices[m.cursor])
		return m.goToTargetFormatSelect(false), nil

	case stateSelectTargetFormat:
		m.targetFormat = converter.NormalizeFormat(m.choices[m.cursor])
		return m.goToFileBrowser(), nil

	case stateFileBrowser:
		if m.cursor < len(m.browserItems) {
			item := m.browserItems[m.cursor]
			if item.isDir {
				// Klasöre gir
				m.browserDir = item.path
				m.cursor = 0
				m.loadBrowserItems()
				return m, nil
			} else {
				// Dosya seç ve dönüştür
				m.selectedFile = item.path
				m.state = stateConverting
				return m, m.doConvert()
			}
		}

	case stateBatchSelectCategory:
		m.selectedCategory = m.cursor
		return m.goToSourceFormatSelect(true), nil

	case stateBatchSelectSourceFormat:
		m.sourceFormat = converter.NormalizeFormat(m.choices[m.cursor])
		return m.goToTargetFormatSelect(true), nil

	case stateBatchSelectTargetFormat:
		m.targetFormat = converter.NormalizeFormat(m.choices[m.cursor])
		m.state = stateBatchConverting
		return m, m.doBatchConvert()

	case stateConvertDone, stateBatchDone:
		return m.goToMainMenu(), nil
	}

	return m, nil
}

func (m interactiveModel) goToMainMenu() interactiveModel {
	m.state = stateMainMenu
	m.cursor = 0
	m.sourceFormat = ""
	m.targetFormat = ""
	m.selectedFile = ""
	m.selectedCategory = 0
	m.browserItems = nil
	m.resultMsg = ""
	m.resultErr = false
	m.choices = []string{
		"Dosya Dönüştür",
		"Toplu Dönüştür (Batch)",
		"Desteklenen Formatlar",
		"Çıkış",
	}
	m.choiceIcons = []string{"🔄", "📦", "📋", "👋"}
	m.choiceDescs = []string{
		"Tek bir dosyayı başka formata dönüştür",
		"Dizindeki tüm dosyaları toplu dönüştür",
		"Desteklenen format ve dönüşüm yollarını gör",
		"Uygulamadan çık",
	}
	return m
}

func (m interactiveModel) goBack() interactiveModel {
	switch m.state {
	case stateSelectCategory:
		return m.goToMainMenu()
	case stateSelectSourceFormat:
		return m.goToCategorySelect(false)
	case stateSelectTargetFormat:
		return m.goToSourceFormatSelect(false)
	case stateFileBrowser:
		return m.goToTargetFormatSelect(false)
	case stateBatchSelectCategory:
		return m.goToMainMenu()
	case stateBatchSelectSourceFormat:
		return m.goToCategorySelect(true)
	case stateBatchSelectTargetFormat:
		return m.goToSourceFormatSelect(true)
	case stateConvertDone, stateBatchDone, stateFormats:
		return m.goToMainMenu()
	default:
		return m.goToMainMenu()
	}
}

func (m interactiveModel) goToCategorySelect(isBatch bool) interactiveModel {
	m.cursor = 0
	m.choices = make([]string, len(categories))
	m.choiceIcons = make([]string, len(categories))
	m.choiceDescs = make([]string, len(categories))
	for i, cat := range categories {
		m.choices[i] = cat.Name
		m.choiceIcons[i] = cat.Icon
		m.choiceDescs[i] = cat.Desc
	}

	if isBatch {
		m.state = stateBatchSelectCategory
	} else {
		m.state = stateSelectCategory
	}
	return m
}

func (m interactiveModel) goToSourceFormatSelect(isBatch bool) interactiveModel {
	cat := categories[m.selectedCategory]

	allPairs := converter.GetAllConversions()
	catFormatSet := make(map[string]bool)
	for _, f := range cat.Formats {
		catFormatSet[f] = true
	}

	sourceSet := make(map[string]bool)
	for _, p := range allPairs {
		if catFormatSet[p.From] {
			sourceSet[p.From] = true
		}
	}

	var sourceFormats []string
	for f := range sourceSet {
		sourceFormats = append(sourceFormats, f)
	}
	sort.Strings(sourceFormats)

	m.choices = make([]string, len(sourceFormats))
	m.choiceIcons = make([]string, len(sourceFormats))
	m.choiceDescs = nil
	for i, f := range sourceFormats {
		m.choices[i] = strings.ToUpper(f)
		m.choiceIcons[i] = cat.Icon
	}
	m.cursor = 0

	if isBatch {
		m.state = stateBatchSelectSourceFormat
	} else {
		m.state = stateSelectSourceFormat
	}
	return m
}

func (m interactiveModel) goToTargetFormatSelect(isBatch bool) interactiveModel {
	pairs := converter.GetConversionsFrom(m.sourceFormat)
	cat := categories[m.selectedCategory]

	m.choices = make([]string, len(pairs))
	m.choiceIcons = make([]string, len(pairs))
	m.choiceDescs = nil
	for i, p := range pairs {
		m.choices[i] = strings.ToUpper(p.To)
		m.choiceIcons[i] = cat.Icon
	}
	m.cursor = 0

	if isBatch {
		m.state = stateBatchSelectTargetFormat
	} else {
		m.state = stateSelectTargetFormat
	}
	return m
}

func (m *interactiveModel) goToFileBrowser() interactiveModel {
	m.state = stateFileBrowser
	m.cursor = 0
	m.loadBrowserItems()
	return *m
}

func (m *interactiveModel) loadBrowserItems() {
	m.browserItems = nil

	entries, err := os.ReadDir(m.browserDir)
	if err != nil {
		return
	}

	ext := "." + m.sourceFormat

	// Üst dizin (.. )
	parent := filepath.Dir(m.browserDir)
	if parent != m.browserDir {
		m.browserItems = append(m.browserItems, browserEntry{
			name:  ".. (üst dizin)",
			path:  parent,
			isDir: true,
		})
	}

	// Klasörler
	var dirs []browserEntry
	var files []browserEntry

	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue // Gizli dosyaları atla
		}

		fullPath := filepath.Join(m.browserDir, e.Name())

		if e.IsDir() {
			dirs = append(dirs, browserEntry{
				name:  e.Name(),
				path:  fullPath,
				isDir: true,
			})
		} else if strings.ToLower(filepath.Ext(e.Name())) == ext {
			files = append(files, browserEntry{
				name:  e.Name(),
				path:  fullPath,
				isDir: false,
			})
		}
	}

	// Önce klasörler, sonra dosyalar
	m.browserItems = append(m.browserItems, dirs...)
	m.browserItems = append(m.browserItems, files...)
}

func (m interactiveModel) doConvert() tea.Cmd {
	return func() tea.Msg {
		start := time.Now()

		conv, err := converter.FindConverter(m.sourceFormat, m.targetFormat)
		if err != nil {
			return convertDoneMsg{err: err, duration: time.Since(start)}
		}

		// Çıktıyı varsayılan olarak Desktop'a kaydet
		outputPath := converter.BuildOutputPath(m.selectedFile, m.defaultOutput, m.targetFormat, "")
		opts := converter.Options{Quality: 0, Verbose: false}

		// Çıktı dizininin var olduğundan emin ol
		os.MkdirAll(filepath.Dir(outputPath), 0755)

		err = conv.Convert(m.selectedFile, outputPath, opts)
		return convertDoneMsg{
			err:      err,
			duration: time.Since(start),
			output:   outputPath,
		}
	}
}

func (m interactiveModel) doBatchConvert() tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		cwd, _ := os.Getwd()

		ext := "." + m.sourceFormat
		var files []string
		entries, _ := os.ReadDir(cwd)
		for _, e := range entries {
			if !e.IsDir() && strings.ToLower(filepath.Ext(e.Name())) == ext {
				files = append(files, filepath.Join(cwd, e.Name()))
			}
		}

		succeeded := 0
		failed := 0
		total := len(files)

		for _, f := range files {
			conv, err := converter.FindConverter(m.sourceFormat, m.targetFormat)
			if err != nil {
				failed++
				continue
			}

			outputPath := converter.BuildOutputPath(f, m.defaultOutput, m.targetFormat, "")
			opts := converter.Options{Quality: 0, Verbose: false}

			if err := conv.Convert(f, outputPath, opts); err != nil {
				failed++
			} else {
				succeeded++
			}
		}

		return batchDoneMsg{
			total:     total,
			succeeded: succeeded,
			failed:    failed,
			duration:  time.Since(start),
		}
	}
}

// ========================================
// Yardımcı fonksiyonlar
// ========================================

func getHomeDir() string {
	u, err := user.Current()
	if err != nil {
		return "/"
	}
	return u.HomeDir
}

func shortenPath(path string) string {
	home := getHomeDir()
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func gradientText(text string, colors []lipgloss.Color) string {
	if len(colors) == 0 {
		return text
	}
	runes := []rune(text)
	var result strings.Builder
	for i, r := range runes {
		colorIdx := i % len(colors)
		style := lipgloss.NewStyle().Bold(true).Foreground(colors[colorIdx])
		result.WriteString(style.Render(string(r)))
	}
	return result.String()
}

func sortedKeys(m map[string]bool) []string {
	var keys []string
	for k := range m {
		keys = append(keys, strings.ToUpper(k))
	}
	sort.Strings(keys)
	return keys
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fµs", float64(d.Microseconds()))
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Milliseconds()))
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// ========================================
// Giriş noktası
// viewDependencies sistem bağımlılıklarını gösterir
func (m interactiveModel) viewDependencies() string {
	var b strings.Builder

	b.WriteString(bannerStyle.Render("SİSTEM KONTROLÜ & BAĞIMLILIKLAR"))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("Bu araçların kurulu olması daha kaliteli dönüşüm sağlar."))
	b.WriteString("\n\n")

	// Başlık
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%-15s %-10s %-35s %s", "ARAÇ", "DURUM", "YOL", "VERSİYON")))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("-", 80)))
	b.WriteString("\n")

	for _, tool := range m.dependencies {
		status := "❌ Yok"
		statusStyle := errorStyle
		if tool.Available {
			status = "✅ Var"
			statusStyle = successStyle
		}

		path := tool.Path
		if len(path) > 35 {
			path = "..." + path[len(path)-32:]
		}
		if path == "" {
			path = "-"
		}

		ver := tool.Version
		if ver == "" {
			ver = "-"
		}

		line := fmt.Sprintf("%-15s %-10s %-35s %s",
			tool.Name,
			status,
			path,
			ver,
		)

		if tool.Available {
			b.WriteString(statusStyle.Render(line))
		} else {
			b.WriteString(dimStyle.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("ESC: Geri dön"))

	return b.String()
}

// ========================================

func RunInteractive() error {
	deps := converter.CheckDependencies()
	p := tea.NewProgram(newInteractiveModel(deps), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
