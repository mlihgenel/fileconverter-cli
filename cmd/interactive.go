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

	"github.com/mlihgenel/fileconverter-cli/internal/config"
	"github.com/mlihgenel/fileconverter-cli/internal/converter"
	"github.com/mlihgenel/fileconverter-cli/internal/installer"
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
	{Name: "Belgeler", Icon: "", Desc: "MD, HTML, PDF, DOCX, TXT, ODT, RTF, CSV", Formats: []string{"md", "html", "pdf", "docx", "txt", "odt", "rtf", "csv"}},
	{Name: "Ses Dosyaları", Icon: "", Desc: "MP3, WAV, OGG, FLAC, AAC, M4A, WMA, OPUS, WEBM", Formats: []string{"mp3", "wav", "ogg", "flac", "aac", "m4a", "wma", "opus", "webm"}},
	{Name: "Görseller", Icon: "", Desc: "PNG, JPEG, WEBP, BMP, GIF, TIFF, ICO", Formats: []string{"png", "jpg", "webp", "bmp", "gif", "tif", "ico"}},
	{Name: "Video Dosyaları", Icon: "", Desc: "MP4, MOV, MKV, AVI, WEBM, M4V, WMV, FLV (GIF'e dönüştürme dahil)", Formats: []string{"mp4", "mov", "mkv", "avi", "webm", "m4v", "wmv", "flv"}},
}

// ========================================
// State Machine
// ========================================

type screenState int

const (
	stateWelcomeIntro screenState = iota
	stateWelcomeDeps
	stateWelcomeInstalling
	stateMainMenu
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
	stateSettings
	stateSettingsBrowser
	stateMissingDep
	stateMissingDepInstalling
	stateBatchBrowser
	stateResizeConfig
	stateResizePresetSelect
	stateResizeManualWidth
	stateResizeManualHeight
	stateResizeManualUnit
	stateResizeManualDPI
	stateResizeModeSelect
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
	categoryIndices  []int

	// Akış tipi
	flowIsBatch    bool
	flowResizeOnly bool

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

	// Karşılama ekranı
	isFirstRun         bool
	welcomeCharIdx     int
	showCursor         bool
	installingToolName string
	installResult      string

	// Dönüşüm öncesi bağımlılık kontrolü
	pendingConvertCmd  tea.Cmd
	missingDepName     string
	missingDepToolName string
	isBatchPending     bool

	// Ayarlar
	settingsBrowserDir   string
	settingsBrowserItems []browserEntry

	// Boyutlandırma
	resizeIsBatchFlow   bool
	resizeSpec          *converter.ResizeSpec
	resizeMethod        string
	resizePresetList    []converter.ResizePreset
	resizePresetName    string
	resizeModeName      string
	resizeWidthInput    string
	resizeHeightInput   string
	resizeUnit          string
	resizeDPIInput      string
	resizeValidationErr string
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

type installDoneMsg struct {
	err error
}

type tickMsg time.Time

func newInteractiveModel(deps []converter.ExternalTool, firstRun bool) interactiveModel {
	homeDir := getHomeDir()

	initialState := stateMainMenu
	if firstRun {
		initialState = stateWelcomeIntro
	}

	// Varsayılan çıktı dizinini config'den oku
	outputDir := config.GetDefaultOutputDir()
	if outputDir == "" {
		outputDir = filepath.Join(homeDir, "Desktop")
	}

	return interactiveModel{
		state:  initialState,
		cursor: 0,
		choices: []string{
			"Dosya Dönüştür",
			"Toplu Dönüştür (Batch)",
			"Boyutlandır",
			"Toplu Boyutlandır",
			"Desteklenen Formatlar",
			"Sistem Kontrolü",
			"Ayarlar",
			"Çıkış",
		},
		choiceIcons: []string{"", "", "", "", "", "", "", ""},
		choiceDescs: []string{
			"Tek bir dosyayı başka formata dönüştür",
			"Dizindeki tüm dosyaları toplu dönüştür",
			"Tek dosya için görsel/video boyutlandırma",
			"Dizindeki dosyalar için toplu boyutlandırma",
			"Desteklenen format ve dönüşüm yollarını gör",
			"Harici araçların (FFmpeg, LibreOffice, Pandoc) durumu",
			"Varsayılan çıktı dizini ve tercihler",
			"Uygulamadan çık",
		},
		browserDir:     outputDir,
		defaultOutput:  outputDir,
		width:          80,
		height:         24,
		dependencies:   deps,
		isFirstRun:     firstRun,
		showCursor:     true,
		resizeMethod:   "none",
		resizeModeName: "pad",
		resizeUnit:     "px",
		resizeDPIInput: "96",
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
		// Spinner animasyonu
		if m.state == stateConverting || m.state == stateBatchConverting || m.state == stateWelcomeInstalling || m.state == stateMissingDepInstalling {
			m.spinnerTick++
			m.spinnerIdx = m.spinnerTick % len(spinnerFrames)
			// Progress bar pulsing efekti
			if m.spinnerTick%5 == 0 {
				m.showCursor = !m.showCursor
			}
		}

		// Karşılama ekranı typing animasyonu
		if m.state == stateWelcomeIntro {
			// Her tick'te 2 karakter ekle
			totalDesiredChars := 0
			for _, line := range welcomeDescLines {
				totalDesiredChars += len([]rune(line))
			}
			if m.welcomeCharIdx < totalDesiredChars {
				m.welcomeCharIdx += 2
				if m.welcomeCharIdx > totalDesiredChars {
					m.welcomeCharIdx = totalDesiredChars
				}
			}
			// Yanıp sönen cursor
			if m.spinnerTick%5 == 0 {
				m.showCursor = !m.showCursor
			}
		}

		// Bağımlılık ekranında cursor yanıp sönme
		if m.state == stateWelcomeDeps {
			if m.spinnerTick%5 == 0 {
				m.showCursor = !m.showCursor
			}
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

	case installDoneMsg:
		// Bağımlılıkları yeniden kontrol et
		m.dependencies = converter.CheckDependencies()

		if m.state == stateMissingDepInstalling {
			// Dönüşüm öncesi kurulumdan geliyoruz
			if msg.err != nil {
				m.resultMsg = fmt.Sprintf("HATA: %s kurulamadı: %s", m.missingDepToolName, msg.err.Error())
				m.resultErr = true
				m.state = stateConvertDone
				return m, nil
			}
			// Kurulum başarılı — dönüşüme devam et
			if m.isBatchPending {
				m.state = stateBatchConverting
			} else {
				m.state = stateConverting
			}
			return m, m.pendingConvertCmd
		}

		// Welcome ekranından geliyoruz
		if msg.err != nil {
			m.installResult = fmt.Sprintf("HATA: Kurulum hatasi: %s", msg.err.Error())
		} else {
			m.installResult = "Kurulum tamamlandi."
		}
		config.MarkFirstRunDone()
		m.state = stateWelcomeDeps
		m.cursor = 0
		return m, nil

	case tea.KeyMsg:
		// Karşılama ekranında "q" çıkmaya yönlendirmesin
		if m.state == stateWelcomeIntro || m.state == stateWelcomeDeps || m.state == stateWelcomeInstalling {
			switch msg.String() {
			case "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "enter":
				return m.handleEnter()
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				max := m.getMaxCursor()
				if m.cursor < max {
					m.cursor++
				}
			}
			return m, nil
		}

		if m.isResizeTextInputState() {
			switch msg.String() {
			case "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "q":
				return m.goToMainMenu(), nil
			case "enter":
				return m.handleEnter()
			case "esc":
				return m.goBack(), nil
			case "backspace", "ctrl+h":
				m.popResizeInput()
				return m, nil
			default:
				if m.appendResizeInput(msg.String()) {
					return m, nil
				}
				return m, nil
			}
		}

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
	case stateWelcomeIntro:
		return 0
	case stateWelcomeDeps:
		return 1
	case stateSettings:
		return 1
	case stateMissingDep:
		return 1
	case stateSettingsBrowser:
		return len(m.settingsBrowserItems) // +1 for "Bu dizini seç" button
	case stateBatchBrowser:
		// Klasör sayısı + 1 ("Dönüştür" butonu)
		dirCount := 0
		for _, item := range m.browserItems {
			if item.isDir {
				dirCount++
			}
		}
		return dirCount // dirCount = son klasör indexı + 1 (dönüştür butonu)
	case stateResizeManualWidth, stateResizeManualHeight, stateResizeManualDPI:
		return 0
	default:
		return len(m.choices) - 1
	}
}

func (m interactiveModel) View() string {
	if m.quitting {
		return gradientText("  Cikis yapiliyor", gradientColors) + "\n\n"
	}

	switch m.state {
	case stateWelcomeIntro:
		return m.viewWelcomeIntro()
	case stateWelcomeDeps:
		return m.viewWelcomeDeps()
	case stateWelcomeInstalling:
		return m.viewWelcomeInstalling()
	case stateMainMenu:
		return m.viewMainMenu()
	case stateSelectCategory:
		if m.flowResizeOnly {
			return m.viewSelectCategory("Boyutlandırma — Dosya türü seçin:")
		}
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
		if m.flowResizeOnly {
			return m.viewSelectCategory("Toplu Boyutlandırma — Dosya türü seçin:")
		}
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
	case stateSettings:
		return m.viewSettings()
	case stateSettingsBrowser:
		return m.viewSettingsBrowser()
	case stateMissingDep:
		return m.viewMissingDep()
	case stateMissingDepInstalling:
		return m.viewMissingDepInstalling()
	case stateBatchBrowser:
		return m.viewBatchBrowser()
	case stateResizeConfig:
		return m.viewResizeConfig()
	case stateResizePresetSelect:
		return m.viewResizePresetSelect()
	case stateResizeManualWidth:
		return m.viewResizeNumericInput("Manuel Genişlik", m.resizeWidthInput, "Örnek: 1080")
	case stateResizeManualHeight:
		return m.viewResizeNumericInput("Manuel Yükseklik", m.resizeHeightInput, "Örnek: 1920")
	case stateResizeManualUnit:
		return m.viewResizeUnitSelect()
	case stateResizeManualDPI:
		return m.viewResizeNumericInput("DPI Değeri", m.resizeDPIInput, "Örnek: 300 (cm için önerilir)")
	case stateResizeModeSelect:
		return m.viewResizeModeSelect()
	default:
		return ""
	}
}

// ========================================
// Ekranlar
// ========================================

func (m interactiveModel) viewMainMenu() string {
	var b strings.Builder

	// Welcome ekranındaki gradient ASCII art
	for i, line := range welcomeArt {
		colorIdx := i % len(welcomeGradient)
		style := lipgloss.NewStyle().Bold(true).Foreground(welcomeGradient[colorIdx])
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	// Versiyon bilgisi
	versionLine := fmt.Sprintf("             v%s  •  Yerel & Güvenli Dönüştürücü", appVersion)
	b.WriteString(lipgloss.NewStyle().Foreground(dimTextColor).Italic(true).Render(versionLine))
	b.WriteString("\n")

	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" ◆ Ana Menü "))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		icon := ""
		if i < len(m.choiceIcons) {
			icon = m.choiceIcons[i]
		}
		desc := ""
		if i < len(m.choiceDescs) {
			desc = m.choiceDescs[i]
		}
		label := menuLine(icon, choice)

		if i == m.cursor {
			b.WriteString(selectedItemStyle.Render("▸ " + label))
			b.WriteString("\n")
			if desc != "" {
				b.WriteString(lipgloss.NewStyle().PaddingLeft(7).Foreground(dimTextColor).Italic(true).Render(desc))
				b.WriteString("\n")
			}
		} else {
			b.WriteString(normalItemStyle.Render("  " + label))
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

	indices := m.categoryIndices
	if len(indices) == 0 {
		indices = make([]int, len(categories))
		for i := range categories {
			indices[i] = i
		}
	}

	for i, catIdx := range indices {
		cat := categories[catIdx]
		if i == m.cursor {
			// Seçili kategori — kart stili
			card := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(secondaryColor).
				Padding(0, 2).
				MarginLeft(2).
				Width(50)

			content := fmt.Sprintf("%s\n%s",
				lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render(cat.Name),
				descStyle.Render(cat.Desc))

			b.WriteString(card.Render(content))
		} else {
			b.WriteString(normalItemStyle.Render("  " + cat.Name))
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
	crumb := fmt.Sprintf("  %s", cat.Name)
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
		label := menuLine(icon, choice)
		if i == m.cursor {
			b.WriteString(selectedItemStyle.Render("▸ " + label))
		} else {
			b.WriteString(normalItemStyle.Render("  " + label))
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
	crumb := fmt.Sprintf("  %s › %s › %s",
		cat.Name,
		lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render(strings.ToUpper(m.sourceFormat)),
		lipgloss.NewStyle().Bold(true).Foreground(accentColor).Render(strings.ToUpper(m.targetFormat)))
	b.WriteString(breadcrumbStyle.Render(crumb))
	b.WriteString("\n\n")

	b.WriteString(menuTitleStyle.Render(" ◆ Dosya Seçin "))
	b.WriteString("\n")

	// Mevcut dizin
	shortDir := shortenPath(m.browserDir)
	b.WriteString(pathStyle.Render(fmt.Sprintf("  Dizin: %s", shortDir)))
	b.WriteString("\n\n")

	if len(m.browserItems) == 0 {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Bu dizinde .%s dosyası veya klasör bulunamadı!", converter.FormatFilterLabel(m.sourceFormat))))
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
				b.WriteString(selectedItemStyle.Render(fmt.Sprintf("▸ %s/", item.name)))
			} else {
				b.WriteString(normalItemStyle.Render(fmt.Sprintf("  %s/", folderStyle.Render(item.name))))
			}
		} else {
			// Dosyalar
			if i == m.cursor {
				b.WriteString(selectedFileStyle.Render(fmt.Sprintf("▸ %s", item.name)))
			} else {
				b.WriteString(normalItemStyle.Render(fmt.Sprintf("  %s", item.name)))
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
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Çıktı: %s", shortenPath(m.defaultOutput))))
	b.WriteString("\n")
	if m.resizeSpec != nil {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Boyutlandırma: %s", m.resizeSummary())))
		b.WriteString("\n")
	}

	return b.String()
}

func (m interactiveModel) viewConverting() string {
	var b strings.Builder
	b.WriteString("\n\n")

	// Başlık
	frame := spinnerFrames[m.spinnerIdx]
	spinnerStyleLocal := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)

	b.WriteString(spinnerStyleLocal.Render(fmt.Sprintf("  %s Dönüştürülüyor", frame)))

	dots := strings.Repeat(".", (m.spinnerTick/3)%4)
	b.WriteString(dimStyle.Render(dots))
	b.WriteString("\n\n")

	if m.selectedFile != "" {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %s -> %s",
			filepath.Base(m.selectedFile),
			strings.ToUpper(m.targetFormat))))
		b.WriteString("\n\n")
	}

	// Animated progress bar
	barWidth := 40
	// Simüle edilen ilerleme — tick bazlı (0-100 arası)
	progress := m.spinnerTick * 3
	if progress > 95 {
		progress = 95 // Tamamlanana kadar %95'te bekle
	}

	filled := barWidth * progress / 100
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	// Gradient progress bar karakterleri
	var barStr strings.Builder
	for i := 0; i < filled; i++ {
		// Gradient efekti: soldan sağa renk geçişi
		colorIdx := i * len(gradientColors) / barWidth
		if colorIdx >= len(gradientColors) {
			colorIdx = len(gradientColors) - 1
		}
		charStyle := lipgloss.NewStyle().Foreground(gradientColors[colorIdx])
		barStr.WriteString(charStyle.Render("█"))
	}
	// Pulsing head karakter
	if filled < barWidth && filled > 0 {
		if m.showCursor {
			barStr.WriteString(lipgloss.NewStyle().Bold(true).Foreground(accentColor).Render("▓"))
			empty--
		} else {
			barStr.WriteString(lipgloss.NewStyle().Foreground(dimTextColor).Render("░"))
			empty--
		}
	}
	for i := 0; i < empty; i++ {
		barStr.WriteString(lipgloss.NewStyle().Foreground(dimTextColor).Render("░"))
	}

	// Progress bar çerçevesi
	b.WriteString(lipgloss.NewStyle().Foreground(dimTextColor).Render("  ["))
	b.WriteString(barStr.String())
	b.WriteString(lipgloss.NewStyle().Foreground(dimTextColor).Render("] "))

	// Yüzde
	percentStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
	b.WriteString(percentStyle.Render(fmt.Sprintf("%d%%", progress)))
	b.WriteString("\n\n")

	// Alt bilgi
	b.WriteString(dimStyle.Render("  Islem devam ediyor, lütfen bekleyin..."))
	b.WriteString("\n")

	// Cursor blink (progress bar animasyonu için)
	if m.spinnerTick%5 == 0 {
		// showCursor toggle handled in Update
	}

	return b.String()
}

func (m interactiveModel) viewConvertDone() string {
	var b strings.Builder

	b.WriteString("\n")
	if m.resultErr {
		content := errorStyle.Render("  Donusum Basarisiz") + "\n\n"
		content += fmt.Sprintf("  Hata: %s", m.resultMsg)
		b.WriteString(resultBoxStyle.Render(content))
	} else {
		content := successStyle.Render("  Donusum Tamamlandi") + "\n\n"
		content += fmt.Sprintf("  Cikti: %s\n", shortenPath(m.resultMsg))
		content += fmt.Sprintf("  Sure:  %s", formatDuration(m.duration))
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

	content := successStyle.Render("  Toplu Donusum Tamamlandi") + "\n\n"
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

	docFormats := map[string]bool{"md": true, "html": true, "pdf": true, "docx": true, "txt": true, "odt": true, "rtf": true, "csv": true}
	audioFormats := map[string]bool{"mp3": true, "wav": true, "ogg": true, "flac": true, "aac": true, "m4a": true, "wma": true, "opus": true, "webm": true}
	imgFormats := map[string]bool{"png": true, "jpg": true, "webp": true, "bmp": true, "gif": true, "tif": true, "ico": true}
	videoFormats := map[string]bool{"mp4": true, "mov": true, "mkv": true, "avi": true, "webm": true, "m4v": true, "wmv": true, "flv": true, "gif": true}

	// Belge formatları
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("  Belge Formatlari"))
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
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("  Ses Formatlari"))
	if !converter.IsFFmpegAvailable() {
		b.WriteString(errorStyle.Render("  FFmpeg gerekli"))
	}
	b.WriteString("\n")
	audioList := sortedKeys(audioFormats)
	b.WriteString(fmt.Sprintf("     %s\n", dimStyle.Render(strings.Join(audioList, " ↔ ")+"  (çapraz)")))

	// Görsel
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("  Gorsel Formatlari"))
	b.WriteString("\n")
	imgList := sortedKeys(imgFormats)
	b.WriteString(fmt.Sprintf("     %s\n", dimStyle.Render(strings.Join(imgList, " ↔ ")+"  (çapraz)")))

	// Video
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("  Video Formatlari"))
	if !converter.IsFFmpegAvailable() {
		b.WriteString(errorStyle.Render("  FFmpeg gerekli"))
	}
	b.WriteString("\n")
	videoList := sortedKeys(videoFormats)
	b.WriteString(fmt.Sprintf("     %s\n", dimStyle.Render(strings.Join(videoList, " ↔ ")+"  (GIF dahil)")))

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
	case stateWelcomeIntro:
		// Typing animasyonunu atla veya devam et
		totalDesiredChars := 0
		for _, line := range welcomeDescLines {
			totalDesiredChars += len([]rune(line))
		}
		if m.welcomeCharIdx < totalDesiredChars {
			// Animasyonu hızla bitir
			m.welcomeCharIdx = totalDesiredChars
			return m, nil
		}
		// Bağımlılık kontrol ekranına geç
		m.state = stateWelcomeDeps
		m.cursor = 0
		return m, nil

	case stateWelcomeDeps:
		// Eksik araç var mı kontrol et
		hasMissing := false
		for _, dep := range m.dependencies {
			if !dep.Available {
				hasMissing = true
				break
			}
		}

		pm := installer.DetectPackageManager()

		if hasMissing && pm != "" {
			if m.cursor == 0 {
				// Otomatik kur
				m.state = stateWelcomeInstalling
				return m, m.doInstallMissing()
			}
			// Atla
			config.MarkFirstRunDone()
			return m.goToMainMenu(), nil
		}

		// Eksik yok veya PM yok — devam et
		config.MarkFirstRunDone()
		return m.goToMainMenu(), nil

	case stateMainMenu:
		switch m.cursor {
		case 0:
			return m.goToCategorySelect(false, false), nil
		case 1:
			return m.goToCategorySelect(true, false), nil
		case 2:
			return m.goToCategorySelect(false, true), nil
		case 3:
			return m.goToCategorySelect(true, true), nil
		case 4:
			m.state = stateFormats
			m.cursor = 0
			return m, nil
		case 5:
			m.state = stateDependencies
			m.cursor = 0
			return m, nil
		case 6:
			// Ayarlar
			m.state = stateSettings
			m.cursor = 0
			return m, nil
		case 7:
			m.quitting = true
			return m, tea.Quit
		}

	case stateSelectCategory:
		if m.cursor >= 0 && m.cursor < len(m.categoryIndices) {
			m.selectedCategory = m.categoryIndices[m.cursor]
		} else {
			m.selectedCategory = m.cursor
		}
		return m.goToSourceFormatSelect(false), nil

	case stateSelectSourceFormat:
		m.sourceFormat = converter.NormalizeFormat(m.choices[m.cursor])
		m.resetResizeState()
		return m.goToTargetFormatSelect(false), nil

	case stateSelectTargetFormat:
		m.targetFormat = converter.NormalizeFormat(m.choices[m.cursor])
		if m.flowResizeOnly {
			return m.goToResizeConfig(false), nil
		}
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
				// Bağımlılık kontrolü yap
				if depName, toolName := m.checkRequiredDep(); depName != "" {
					m.missingDepName = depName
					m.missingDepToolName = toolName
					m.pendingConvertCmd = m.doConvert()
					m.isBatchPending = false
					m.state = stateMissingDep
					m.cursor = 0
					return m, nil
				}
				m.state = stateConverting
				return m, m.doConvert()
			}
		}

	case stateBatchSelectCategory:
		if m.cursor >= 0 && m.cursor < len(m.categoryIndices) {
			m.selectedCategory = m.categoryIndices[m.cursor]
		} else {
			m.selectedCategory = m.cursor
		}
		return m.goToSourceFormatSelect(true), nil

	case stateBatchSelectSourceFormat:
		m.sourceFormat = converter.NormalizeFormat(m.choices[m.cursor])
		m.resetResizeState()
		return m.goToTargetFormatSelect(true), nil

	case stateBatchSelectTargetFormat:
		m.targetFormat = converter.NormalizeFormat(m.choices[m.cursor])
		if m.flowResizeOnly {
			return m.goToResizeConfig(true), nil
		}
		return m.goToBatchBrowserOrDependencyCheck()

	case stateResizeConfig:
		switch m.cursor {
		case 0:
			m.resizeMethod = "none"
			m.resizeSpec = nil
			m.resizeValidationErr = ""
			return m.proceedAfterResizeSelection()
		case 1:
			m.resizeMethod = "preset"
			return m.goToResizePresetSelect(), nil
		case 2:
			m.resizeMethod = "manual"
			return m.goToResizeManualWidth(), nil
		}

	case stateResizePresetSelect:
		if m.cursor >= 0 && m.cursor < len(m.resizePresetList) {
			m.resizePresetName = m.resizePresetList[m.cursor].Name
			return m.goToResizeModeSelect(), nil
		}

	case stateResizeManualWidth:
		if _, err := parseResizeInputValue(m.resizeWidthInput); err != nil {
			m.resizeValidationErr = "Geçersiz genişlik değeri"
			return m, nil
		}
		m.resizeValidationErr = ""
		return m.goToResizeManualHeight(), nil

	case stateResizeManualHeight:
		if _, err := parseResizeInputValue(m.resizeHeightInput); err != nil {
			m.resizeValidationErr = "Geçersiz yükseklik değeri"
			return m, nil
		}
		m.resizeValidationErr = ""
		return m.goToResizeManualUnitSelect(), nil

	case stateResizeManualUnit:
		if m.cursor == 0 {
			m.resizeUnit = "px"
			return m.goToResizeModeSelect(), nil
		}
		m.resizeUnit = "cm"
		if strings.TrimSpace(m.resizeDPIInput) == "" {
			m.resizeDPIInput = "96"
		}
		return m.goToResizeManualDPI(), nil

	case stateResizeManualDPI:
		if _, err := parseResizeInputValue(m.resizeDPIInput); err != nil {
			m.resizeValidationErr = "Geçersiz DPI değeri"
			return m, nil
		}
		m.resizeValidationErr = ""
		return m.goToResizeModeSelect(), nil

	case stateResizeModeSelect:
		if m.cursor >= 0 && m.cursor < len(resizeModeOptions) {
			m.resizeModeName = resizeModeOptions[m.cursor].Key
		}
		spec, err := m.buildResizeSpecFromSelection()
		if err != nil {
			m.resizeValidationErr = err.Error()
			return m, nil
		}
		m.resizeSpec = spec
		m.resizeValidationErr = ""
		return m.proceedAfterResizeSelection()

	case stateBatchBrowser:
		// Klasör listesinden sayı al
		dirItems := []browserEntry{}
		for _, item := range m.browserItems {
			if item.isDir {
				dirItems = append(dirItems, item)
			}
		}
		if m.cursor < len(dirItems) {
			// Klasöre gir
			m.browserDir = dirItems[m.cursor].path
			m.loadBrowserItems()
			m.cursor = 0
			return m, nil
		}
		// "Dönüştür" butonu
		m.state = stateBatchConverting
		return m, m.doBatchConvert()

	case stateMissingDep:
		if m.cursor == 0 {
			// Kur
			m.state = stateMissingDepInstalling
			m.installingToolName = m.missingDepToolName
			return m, m.doInstallSingleTool(m.missingDepToolName)
		}
		// İptal
		return m.goToMainMenu(), nil

	case stateMissingDepInstalling:
		// Kurulum tamamlandı (installDoneMsg tarafından yönetilecek)
		return m, nil

	case stateSettings:
		switch m.cursor {
		case 0:
			// Varsayılan dizin değiştir
			m.settingsBrowserDir = m.defaultOutput
			m.loadSettingsBrowserItems()
			m.state = stateSettingsBrowser
			m.cursor = 0
			return m, nil
		case 1:
			// Geri
			return m.goToMainMenu(), nil
		}

	case stateSettingsBrowser:
		if m.cursor < len(m.settingsBrowserItems) {
			item := m.settingsBrowserItems[m.cursor]
			if item.isDir {
				m.settingsBrowserDir = item.path
				m.cursor = 0
				m.loadSettingsBrowserItems()
				return m, nil
			}
		} else if m.cursor == len(m.settingsBrowserItems) {
			// "Bu dizini seç" butonu
			m.defaultOutput = m.settingsBrowserDir
			config.SetDefaultOutputDir(m.settingsBrowserDir)
			m.state = stateSettings
			m.cursor = 0
			return m, nil
		}

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
	m.pendingConvertCmd = nil
	m.missingDepName = ""
	m.missingDepToolName = ""
	m.categoryIndices = nil
	m.flowIsBatch = false
	m.flowResizeOnly = false
	m.resetResizeState()
	m.choices = []string{
		"Dosya Dönüştür",
		"Toplu Dönüştür (Batch)",
		"Boyutlandır",
		"Toplu Boyutlandır",
		"Desteklenen Formatlar",
		"Sistem Kontrolü",
		"Ayarlar",
		"Çıkış",
	}
	m.choiceIcons = []string{"", "", "", "", "", "", "", ""}
	m.choiceDescs = []string{
		"Tek bir dosyayı başka formata dönüştür",
		"Dizindeki tüm dosyaları toplu dönüştür",
		"Tek dosya için görsel/video boyutlandırma",
		"Dizindeki dosyalar için toplu boyutlandırma",
		"Desteklenen format ve dönüşüm yollarını gör",
		"Harici araçların (FFmpeg, LibreOffice, Pandoc) durumu",
		"Varsayılan çıktı dizini ve tercihler",
		"Uygulamadan çık",
	}
	return m
}

func (m interactiveModel) goBack() interactiveModel {
	switch m.state {
	case stateSelectCategory:
		return m.goToMainMenu()
	case stateSelectSourceFormat:
		return m.goToCategorySelect(false, m.flowResizeOnly)
	case stateSelectTargetFormat:
		return m.goToSourceFormatSelect(false)
	case stateFileBrowser:
		if m.flowResizeOnly {
			return m.goToResizeConfig(false)
		}
		return m.goToTargetFormatSelect(false)
	case stateBatchSelectCategory:
		return m.goToMainMenu()
	case stateBatchSelectSourceFormat:
		return m.goToCategorySelect(true, m.flowResizeOnly)
	case stateBatchSelectTargetFormat:
		return m.goToSourceFormatSelect(true)
	case stateBatchBrowser:
		if m.flowResizeOnly {
			return m.goToResizeConfig(true)
		}
		return m.goToTargetFormatSelect(true)
	case stateResizeConfig:
		return m.goToTargetFormatSelect(m.resizeIsBatchFlow)
	case stateResizePresetSelect:
		return m.goToResizeConfig(m.resizeIsBatchFlow)
	case stateResizeManualWidth:
		return m.goToResizeConfig(m.resizeIsBatchFlow)
	case stateResizeManualHeight:
		return m.goToResizeManualWidth()
	case stateResizeManualUnit:
		return m.goToResizeManualHeight()
	case stateResizeManualDPI:
		return m.goToResizeManualUnitSelect()
	case stateResizeModeSelect:
		if m.resizeMethod == "preset" {
			return m.goToResizePresetSelect()
		}
		if m.resizeMethod == "manual" {
			if m.resizeUnit == "cm" {
				return m.goToResizeManualDPI()
			}
			return m.goToResizeManualUnitSelect()
		}
		return m.goToResizeConfig(m.resizeIsBatchFlow)
	case stateConvertDone, stateBatchDone, stateFormats:
		return m.goToMainMenu()
	case stateSettings:
		return m.goToMainMenu()
	case stateSettingsBrowser:
		m.state = stateSettings
		m.cursor = 0
		return m
	case stateMissingDep:
		return m.goToMainMenu()
	default:
		return m.goToMainMenu()
	}
}

func (m interactiveModel) goToCategorySelect(isBatch bool, resizeOnly bool) interactiveModel {
	m.flowIsBatch = isBatch
	m.flowResizeOnly = resizeOnly
	m.cursor = 0

	m.categoryIndices = nil
	for i, cat := range categories {
		if resizeOnly {
			// Boyutlandırma akışında sadece görsel/video kategorilerini göster.
			if len(cat.Formats) == 0 || !converter.IsResizableFormat(cat.Formats[0]) {
				continue
			}
		}
		m.categoryIndices = append(m.categoryIndices, i)
	}

	m.choices = make([]string, len(m.categoryIndices))
	m.choiceIcons = make([]string, len(m.categoryIndices))
	m.choiceDescs = make([]string, len(m.categoryIndices))
	for i, catIdx := range m.categoryIndices {
		cat := categories[catIdx]
		m.choices[i] = cat.Name
		m.choiceIcons[i] = ""
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

	targets := make([]string, 0, len(pairs)+1)
	for _, p := range pairs {
		targets = append(targets, p.To)
	}
	if m.flowResizeOnly && converter.IsResizableFormat(m.sourceFormat) {
		exists := false
		for _, t := range targets {
			if t == m.sourceFormat {
				exists = true
				break
			}
		}
		if !exists {
			targets = append(targets, m.sourceFormat)
		}
	}
	sort.Strings(targets)

	m.choices = make([]string, len(targets))
	m.choiceIcons = make([]string, len(targets))
	m.choiceDescs = nil
	for i, target := range targets {
		m.choices[i] = strings.ToUpper(target)
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
		} else if converter.HasFormatExtension(e.Name(), m.sourceFormat) {
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
		opts := converter.Options{Quality: 0, Verbose: false, Resize: m.resizeSpec}

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
	scanDir := m.browserDir
	if scanDir == "" {
		scanDir = m.defaultOutput
	}
	return func() tea.Msg {
		start := time.Now()

		var files []string
		entries, _ := os.ReadDir(scanDir)
		for _, e := range entries {
			if !e.IsDir() && converter.HasFormatExtension(e.Name(), m.sourceFormat) {
				files = append(files, filepath.Join(scanDir, e.Name()))
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
			opts := converter.Options{Quality: 0, Verbose: false, Resize: m.resizeSpec}

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

func menuLine(icon string, text string) string {
	if strings.TrimSpace(icon) == "" {
		return text
	}
	return fmt.Sprintf("%s  %s", icon, text)
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
		status := "Yok"
		statusStyle := errorStyle
		if tool.Available {
			status = "Var"
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

// doInstallMissing eksik araçları kurar
func (m interactiveModel) doInstallMissing() tea.Cmd {
	return func() tea.Msg {
		for _, dep := range m.dependencies {
			if !dep.Available {
				_, err := installer.InstallTool(dep.Name)
				if err != nil {
					return installDoneMsg{err: err}
				}
			}
		}
		return installDoneMsg{err: nil}
	}
}

// doInstallSingleTool tek bir aracı kurar
func (m interactiveModel) doInstallSingleTool(toolName string) tea.Cmd {
	return func() tea.Msg {
		_, err := installer.InstallTool(toolName)
		return installDoneMsg{err: err}
	}
}

// checkRequiredDep dönüşüm için gerekli bağımlılığı kontrol eder
// Eksikse (depName, toolName) döner, yoksa ("", "") döner
func (m interactiveModel) checkRequiredDep() (string, string) {
	cat := categories[m.selectedCategory]

	// Ses dönüşümü → FFmpeg
	if cat.Name == "Ses Dosyaları" {
		if !converter.IsFFmpegAvailable() {
			return "FFmpeg", "ffmpeg"
		}
	}

	// Video dönüşümü → FFmpeg
	if cat.Name == "Video Dosyaları" {
		if !converter.IsFFmpegAvailable() {
			return "FFmpeg", "ffmpeg"
		}
	}

	// Belge dönüşümlerinde LibreOffice/Pandoc kontrolü
	if cat.Name == "Belgeler" {
		from := m.sourceFormat
		to := m.targetFormat

		// ODT/RTF dönüşümleri → LibreOffice gerekli
		needsLibreOffice := false
		libreOfficeFormats := map[string]bool{"odt": true, "rtf": true, "xlsx": true}
		if libreOfficeFormats[from] || libreOfficeFormats[to] {
			needsLibreOffice = true
		}
		// CSV → XLSX de LibreOffice gerektirir
		if from == "csv" && to == "xlsx" {
			needsLibreOffice = true
		}
		// DOCX/PDF çapraz dönüşümlerde LibreOffice yardımcı
		if (from == "docx" || from == "pdf") && (to == "odt" || to == "rtf") {
			needsLibreOffice = true
		}

		if needsLibreOffice && !converter.IsLibreOfficeAvailable() {
			return "LibreOffice", "libreoffice"
		}

		// Pandoc kontrolü (md → pdf gibi bazı dönüşümler)
		if (from == "md" && to == "pdf") || (from == "md" && to == "docx") {
			if !converter.IsPandocAvailable() {
				// Pandoc opsiyonel — Go fallback var, ama bilgilendirelim
				// Zorunlu değil, bu yüzden boş dönüyoruz
			}
		}
	}

	return "", ""
}

// loadSettingsBrowserItems ayarlar dizin tarayıcısına öğeleri yükler
func (m *interactiveModel) loadSettingsBrowserItems() {
	entries, err := os.ReadDir(m.settingsBrowserDir)
	if err != nil {
		m.settingsBrowserItems = nil
		return
	}

	var items []browserEntry

	// Üst dizin
	parent := filepath.Dir(m.settingsBrowserDir)
	if parent != m.settingsBrowserDir {
		items = append(items, browserEntry{
			name:  "..",
			path:  parent,
			isDir: true,
		})
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue // Sadece dizinler
		}
		if strings.HasPrefix(e.Name(), ".") {
			continue // Gizli dizinleri atla
		}
		items = append(items, browserEntry{
			name:  e.Name(),
			path:  filepath.Join(m.settingsBrowserDir, e.Name()),
			isDir: true,
		})
	}

	m.settingsBrowserItems = items
}

// ========================================
// Yeni View Fonksiyonları
// ========================================

// viewSettings ayarlar ekranı
func (m interactiveModel) viewSettings() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" Ayarlar "))
	b.WriteString("\n\n")

	// Mevcut varsayılan dizin
	b.WriteString(lipgloss.NewStyle().Foreground(textColor).Render("  Varsayılan çıktı dizini:"))
	b.WriteString("\n")
	b.WriteString(pathStyle.Render("  " + m.defaultOutput))
	b.WriteString("\n\n")

	options := []string{"Varsayilan dizini degistir", "Ana menuye don"}
	for i, opt := range options {
		if i == m.cursor {
			b.WriteString(selectedItemStyle.Render(fmt.Sprintf("▸ %s", opt)))
		} else {
			b.WriteString(normalItemStyle.Render(fmt.Sprintf("  %s", opt)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç  •  Esc Geri"))
	b.WriteString("\n")

	return b.String()
}

// viewSettingsBrowser dizin seçici ekranı
func (m interactiveModel) viewSettingsBrowser() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" Varsayilan Cikti Dizini Sec "))
	b.WriteString("\n\n")

	// Mevcut dizin
	b.WriteString(dimStyle.Render("  Konum: "))
	b.WriteString(pathStyle.Render(m.settingsBrowserDir))
	b.WriteString("\n\n")

	for i, item := range m.settingsBrowserItems {
		if i == m.cursor {
			b.WriteString(selectedItemStyle.Render(fmt.Sprintf("▸ %s", item.name)))
		} else {
			b.WriteString(normalItemStyle.Render(fmt.Sprintf("  %s", item.name)))
		}
		b.WriteString("\n")
	}

	// "Bu dizini seç" butonu
	selectIdx := len(m.settingsBrowserItems)
	b.WriteString("\n")
	if m.cursor == selectIdx {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(accentColor).Render("  ▸ Bu dizini sec"))
	} else {
		b.WriteString(dimStyle.Render("    Bu dizini sec"))
	}
	b.WriteString("\n")

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç/Gir  •  Esc Geri"))
	b.WriteString("\n")

	return b.String()
}

// viewMissingDep eksik bağımlılık uyarısı
func (m interactiveModel) viewMissingDep() string {
	var b strings.Builder

	b.WriteString("\n")

	// Uyarı kutusu
	warningBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(warningColor).
		Padding(1, 3).
		MarginLeft(2).
		Width(60)

	content := fmt.Sprintf(
		"%s kurulu degil!\n\n"+
			"%s olmadan %s → %s dönüşümü yapılamaz.\n\n"+
			"Şimdi kurmak ister misiniz?",
		m.missingDepName,
		m.missingDepName,
		strings.ToUpper(m.sourceFormat),
		strings.ToUpper(m.targetFormat),
	)

	b.WriteString(warningBox.Render(content))
	b.WriteString("\n\n")

	options := []string{
		fmt.Sprintf("%s'i kur", m.missingDepName),
		"Iptal et",
	}
	for i, opt := range options {
		if i == m.cursor {
			b.WriteString(selectedItemStyle.Render(fmt.Sprintf("  ▸ %s", opt)))
		} else {
			b.WriteString(normalItemStyle.Render(fmt.Sprintf("    %s", opt)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Paket yöneticisi bilgisi
	pm := installer.DetectPackageManager()
	if pm != "" {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Paket yöneticisi: %s", pm)))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(warningColor).Render("  Paket yoneticisi bulunamadi — manuel kurulum gerekebilir"))
	}
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç"))
	b.WriteString("\n")

	return b.String()
}

// viewMissingDepInstalling bağımlılık kurulumu sırasında gösterilen ekran
func (m interactiveModel) viewMissingDepInstalling() string {
	var b strings.Builder

	b.WriteString("\n\n")

	frame := spinnerFrames[m.spinnerIdx]
	spinnerStyle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)

	b.WriteString(spinnerStyle.Render(fmt.Sprintf("  %s %s kuruluyor", frame, m.missingDepToolName)))

	dots := strings.Repeat(".", (m.spinnerTick/3)%4)
	b.WriteString(dimStyle.Render(dots))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("  Lütfen bekleyin, kurulum devam ediyor..."))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().Foreground(dimTextColor).Italic(true).Render(
		"  Kurulum tamamlandığında dönüşüm otomatik başlayacak."))
	b.WriteString("\n")

	return b.String()
}

// viewBatchBrowser toplu dönüşüm için dizin seçici
func (m interactiveModel) viewBatchBrowser() string {
	var b strings.Builder

	b.WriteString("\n")

	// Breadcrumb
	cat := categories[m.selectedCategory]
	crumb := fmt.Sprintf("  %s › %s -> %s  (Toplu)",
		cat.Name,
		lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render(strings.ToUpper(m.sourceFormat)),
		lipgloss.NewStyle().Bold(true).Foreground(accentColor).Render(strings.ToUpper(m.targetFormat)))
	b.WriteString(breadcrumbStyle.Render(crumb))
	b.WriteString("\n\n")

	b.WriteString(menuTitleStyle.Render(" Kaynak Dizin Secin "))
	b.WriteString("\n")

	// Mevcut dizin
	shortDir := shortenPath(m.browserDir)
	b.WriteString(pathStyle.Render(fmt.Sprintf("  Dizin: %s", shortDir)))
	b.WriteString("\n\n")

	// Eşleşen dosya sayısı
	fileCount := 0
	for _, item := range m.browserItems {
		if !item.isDir {
			fileCount++
		}
	}

	if fileCount > 0 {
		b.WriteString(successStyle.Render(fmt.Sprintf("  Bu dizinde %d adet .%s dosyasi bulundu", fileCount, converter.FormatFilterLabel(m.sourceFormat))))
	} else {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Bu dizinde .%s dosyasi bulunamadi", converter.FormatFilterLabel(m.sourceFormat))))
	}
	b.WriteString("\n\n")

	// Klasörler (gezinme)
	dirIdx := 0
	for _, item := range m.browserItems {
		if !item.isDir {
			continue
		}
		if dirIdx == m.cursor {
			b.WriteString(selectedItemStyle.Render(fmt.Sprintf("▸ %s/", item.name)))
		} else {
			b.WriteString(normalItemStyle.Render(fmt.Sprintf("  %s/", folderStyle.Render(item.name))))
		}
		b.WriteString("\n")
		dirIdx++
	}

	// "Dönüştür" butonu
	b.WriteString("\n")
	if m.cursor == dirIdx {
		btn := fmt.Sprintf("▸ Bu dizindeki %d dosyayi donustur", fileCount)
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(accentColor).Render("  " + btn))
	} else {
		btn := fmt.Sprintf("  Bu dizindeki %d dosyayi donustur", fileCount)
		b.WriteString(dimStyle.Render("  " + btn))
	}
	b.WriteString("\n")

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç/Gir  •  Esc Geri"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Cikti: %s", shortenPath(m.defaultOutput))))
	b.WriteString("\n")
	if m.resizeSpec != nil {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Boyutlandirma: %s", m.resizeSummary())))
		b.WriteString("\n")
	}

	return b.String()
}

func RunInteractive() error {
	deps := converter.CheckDependencies()
	firstRun := config.IsFirstRun()
	p := tea.NewProgram(newInteractiveModel(deps, firstRun), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
