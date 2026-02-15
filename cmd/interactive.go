package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/melihgenel/fileconverter/internal/converter"
	"github.com/melihgenel/fileconverter/internal/ui"
)

// ========================================
// Stiller
// ========================================

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00D4FF")).
			MarginBottom(1)

	menuTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 2)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00D4FF")).
			PaddingLeft(2)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDDDDD")).
			PaddingLeft(4)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#04B575"))

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF4444"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00AAFF"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2).
			MarginTop(1)
)

// ========================================
// State Machine
// ========================================

type screenState int

const (
	stateMainMenu screenState = iota
	stateSelectSourceFormat
	stateSelectTargetFormat
	stateSelectFile
	stateConverting
	stateConvertDone
	stateBatchSelectSourceFormat
	stateBatchSelectTargetFormat
	stateBatchConverting
	stateBatchDone
	stateFormats
)

// ========================================
// Model
// ========================================

type interactiveModel struct {
	state       screenState
	cursor      int
	choices     []string
	choiceIcons []string

	// Dönüşüm bilgileri
	sourceFormat string
	targetFormat string
	selectedFile string
	files        []string

	// Sonuçlar
	resultMsg string
	resultErr bool
	duration  time.Duration

	// Batch sonuçları
	batchTotal     int
	batchSucceeded int
	batchFailed    int

	// Format tablosu
	formatLines []string

	// Pencere boyutu
	width  int
	height int

	// Çıkış
	quitting bool
}

func newInteractiveModel() interactiveModel {
	return interactiveModel{
		state:  stateMainMenu,
		cursor: 0,
		choices: []string{
			"Dosya Dönüştür",
			"Toplu Dönüştür (Batch)",
			"Desteklenen Formatlar",
			"Çıkış",
		},
		choiceIcons: []string{"📄", "📦", "📋", "👋"},
		width:       80,
		height:      24,
	}
}

// dönüşüm mesajları
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

// ========================================
// bubbletea Interface
// ========================================

func (m interactiveModel) Init() tea.Cmd {
	return nil
}

func (m interactiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

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
		case "ctrl+c", "q":
			if m.state == stateMainMenu {
				m.quitting = true
				return m, tea.Quit
			}
			// Herhangi bir ekrandan ana menüye dön
			return m.goToMainMenu(), nil

		case "esc":
			return m.goBack(), nil

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			max := len(m.choices) - 1
			if m.state == stateSelectFile {
				max = len(m.files) - 1
			} else if m.state == stateFormats {
				return m, nil // Format ekranında navigasyon yok
			}
			if m.cursor < max {
				m.cursor++
			}

		case "enter":
			return m.handleEnter()
		}
	}

	return m, nil
}

func (m interactiveModel) View() string {
	if m.quitting {
		return "\n  👋 Görüşürüz!\n\n"
	}

	switch m.state {
	case stateMainMenu:
		return m.viewMainMenu()
	case stateSelectSourceFormat:
		return m.viewSelectFormat("Kaynak format seçin:", false)
	case stateSelectTargetFormat:
		return m.viewSelectFormat("Hedef format seçin:", true)
	case stateSelectFile:
		return m.viewSelectFile()
	case stateConverting:
		return m.viewConverting()
	case stateConvertDone:
		return m.viewConvertDone()
	case stateBatchSelectSourceFormat:
		return m.viewSelectFormat("Batch — Kaynak format seçin:", false)
	case stateBatchSelectTargetFormat:
		return m.viewSelectFormat("Batch — Hedef format seçin:", true)
	case stateBatchConverting:
		return m.viewConverting()
	case stateBatchDone:
		return m.viewBatchDone()
	case stateFormats:
		return m.viewFormats()
	default:
		return ""
	}
}

// ========================================
// Ekranlar
// ========================================

func (m interactiveModel) viewMainMenu() string {
	var b strings.Builder

	banner := titleStyle.Render(`
  ╔═══════════════════════════════════════════════╗
  ║        FileConverter CLI  v1.0.0              ║
  ║   Yerel dosya format dönüştürücü              ║
  ╚═══════════════════════════════════════════════╝`)

	b.WriteString(banner)
	b.WriteString("\n\n")
	b.WriteString(menuTitleStyle.Render(" Ana Menü "))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		icon := m.choiceIcons[i]
		if i == m.cursor {
			b.WriteString(selectedStyle.Render(fmt.Sprintf("▸ %s  %s", icon, choice)))
		} else {
			b.WriteString(normalStyle.Render(fmt.Sprintf("  %s  %s", icon, choice)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑/↓: Gezin  •  Enter: Seç  •  q: Çıkış"))
	b.WriteString("\n")

	return b.String()
}

func (m interactiveModel) viewSelectFormat(title string, isTarget bool) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(fmt.Sprintf(" %s ", title)))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		icon := ""
		if i < len(m.choiceIcons) {
			icon = m.choiceIcons[i]
		}
		if i == m.cursor {
			b.WriteString(selectedStyle.Render(fmt.Sprintf("▸ %s  %s", icon, choice)))
		} else {
			b.WriteString(normalStyle.Render(fmt.Sprintf("  %s  %s", icon, choice)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.sourceFormat != "" {
		b.WriteString(infoStyle.Render(fmt.Sprintf("  Kaynak format: %s", strings.ToUpper(m.sourceFormat))))
		b.WriteString("\n")
	}
	b.WriteString(dimStyle.Render("  ↑/↓: Gezin  •  Enter: Seç  •  Esc: Geri"))
	b.WriteString("\n")

	return b.String()
}

func (m interactiveModel) viewSelectFile() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(fmt.Sprintf(" %s → %s — Dosya seçin: ",
		strings.ToUpper(m.sourceFormat), strings.ToUpper(m.targetFormat))))
	b.WriteString("\n\n")

	if len(m.files) == 0 {
		b.WriteString(errorStyle.Render("  Bu dizinde ." + m.sourceFormat + " dosyası bulunamadı!"))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("  Esc: Geri"))
		b.WriteString("\n")
		return b.String()
	}

	// Sayfala — her sayfada en fazla 15 dosya göster
	pageSize := 15
	startIdx := 0
	if m.cursor >= pageSize {
		startIdx = m.cursor - pageSize + 1
	}
	endIdx := startIdx + pageSize
	if endIdx > len(m.files) {
		endIdx = len(m.files)
	}

	for i := startIdx; i < endIdx; i++ {
		displayName := filepath.Base(m.files[i])
		if i == m.cursor {
			b.WriteString(selectedStyle.Render(fmt.Sprintf("▸ 📄 %s", displayName)))
		} else {
			b.WriteString(normalStyle.Render(fmt.Sprintf("  📄 %s", displayName)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(infoStyle.Render(fmt.Sprintf("  %d dosya bulundu", len(m.files))))
	if len(m.files) > pageSize {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  (Gösterilen: %d-%d)", startIdx+1, endIdx)))
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑/↓: Gezin  •  Enter: Dönüştür  •  Esc: Geri"))
	b.WriteString("\n")

	return b.String()
}

func (m interactiveModel) viewConverting() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  🔄 Dönüştürülüyor..."))
	b.WriteString("\n")
	return b.String()
}

func (m interactiveModel) viewConvertDone() string {
	var b strings.Builder

	b.WriteString("\n")
	if m.resultErr {
		b.WriteString(boxStyle.Render(
			errorStyle.Render("❌ Dönüşüm Başarısız\n\n") +
				fmt.Sprintf("  Hata: %s", m.resultMsg),
		))
	} else {
		b.WriteString(boxStyle.Render(
			successStyle.Render("✅ Dönüşüm Tamamlandı!\n\n") +
				fmt.Sprintf("  Çıktı: %s\n  Süre:  %s", m.resultMsg, formatDuration(m.duration)),
		))
	}

	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("  Enter/Esc: Ana Menüye Dön"))
	b.WriteString("\n")

	return b.String()
}

func (m interactiveModel) viewBatchDone() string {
	var b strings.Builder

	b.WriteString("\n")

	content := successStyle.Render("🎉 Toplu Dönüşüm Tamamlandı!\n\n")
	content += fmt.Sprintf("  Toplam:    %d dosya\n", m.batchTotal)
	content += successStyle.Render(fmt.Sprintf("  Başarılı:  %d dosya\n", m.batchSucceeded))
	if m.batchFailed > 0 {
		content += errorStyle.Render(fmt.Sprintf("  Başarısız: %d dosya\n", m.batchFailed))
	}
	content += fmt.Sprintf("  Süre:      %s", formatDuration(m.duration))

	b.WriteString(boxStyle.Render(content))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("  Enter/Esc: Ana Menüye Dön"))
	b.WriteString("\n")

	return b.String()
}

func (m interactiveModel) viewFormats() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" Desteklenen Dönüşümler "))
	b.WriteString("\n\n")

	pairs := converter.GetAllConversions()

	// Kategorilere ayır
	docFormats := map[string]bool{"md": true, "html": true, "pdf": true, "docx": true, "txt": true}
	audioFormats := map[string]bool{"mp3": true, "wav": true, "ogg": true, "flac": true, "aac": true, "m4a": true, "wma": true}
	imgFormats := map[string]bool{"png": true, "jpg": true, "webp": true, "bmp": true, "gif": true, "tif": true}

	// Belge formatları
	b.WriteString(infoStyle.Render("  📄 Belge Formatları"))
	b.WriteString("\n")
	for _, p := range pairs {
		if docFormats[p.From] && docFormats[p.To] {
			b.WriteString(fmt.Sprintf("     %s → %s\n",
				lipgloss.NewStyle().Bold(true).Render(strings.ToUpper(p.From)),
				successStyle.Render(strings.ToUpper(p.To))))
		}
	}

	// Ses formatları
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  🎵 Ses Formatları"))
	if !converter.IsFFmpegAvailable() {
		b.WriteString(errorStyle.Render("  (FFmpeg gerekli!)"))
	}
	b.WriteString("\n")
	audioList := []string{}
	for f := range audioFormats {
		audioList = append(audioList, strings.ToUpper(f))
	}
	sort.Strings(audioList)
	b.WriteString(fmt.Sprintf("     %s (çapraz dönüşüm)\n", strings.Join(audioList, " ↔ ")))

	// Görsel formatları
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  🖼️  Görsel Formatları"))
	b.WriteString("\n")
	imgList := []string{}
	for f := range imgFormats {
		imgList = append(imgList, strings.ToUpper(f))
	}
	sort.Strings(imgList)
	b.WriteString(fmt.Sprintf("     %s (çapraz dönüşüm)\n", strings.Join(imgList, " ↔ ")))

	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Toplam: %d dönüşüm yolu", len(pairs))))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("  Esc: Ana Menüye Dön"))
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
		case 0: // Dosya Dönüştür
			return m.goToSourceFormatSelect(false), nil
		case 1: // Toplu Dönüştür
			return m.goToSourceFormatSelect(true), nil
		case 2: // Formatlar
			m.state = stateFormats
			m.cursor = 0
			return m, nil
		case 3: // Çıkış
			m.quitting = true
			return m, tea.Quit
		}

	case stateSelectSourceFormat:
		m.sourceFormat = converter.NormalizeFormat(m.choices[m.cursor])
		return m.goToTargetFormatSelect(false), nil

	case stateSelectTargetFormat:
		m.targetFormat = converter.NormalizeFormat(m.choices[m.cursor])
		return m.goToFileSelect(), nil

	case stateSelectFile:
		if len(m.files) > 0 && m.cursor < len(m.files) {
			m.selectedFile = m.files[m.cursor]
			m.state = stateConverting
			return m, m.doConvert()
		}

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
	m.files = nil
	m.resultMsg = ""
	m.resultErr = false
	m.choices = []string{
		"Dosya Dönüştür",
		"Toplu Dönüştür (Batch)",
		"Desteklenen Formatlar",
		"Çıkış",
	}
	m.choiceIcons = []string{"📄", "📦", "📋", "👋"}
	return m
}

func (m interactiveModel) goBack() interactiveModel {
	switch m.state {
	case stateSelectSourceFormat:
		return m.goToMainMenu()
	case stateSelectTargetFormat:
		return m.goToSourceFormatSelect(false)
	case stateSelectFile:
		return m.goToTargetFormatSelect(false)
	case stateBatchSelectSourceFormat:
		return m.goToMainMenu()
	case stateBatchSelectTargetFormat:
		return m.goToSourceFormatSelect(true)
	case stateConvertDone, stateBatchDone, stateFormats:
		return m.goToMainMenu()
	default:
		return m.goToMainMenu()
	}
}

func (m interactiveModel) goToSourceFormatSelect(isBatch bool) interactiveModel {
	allFormats := getUniqueSourceFormats()

	m.choices = allFormats
	m.choiceIcons = make([]string, len(allFormats))
	for i, f := range allFormats {
		m.choiceIcons[i] = ui.PrintFormatCategory(converter.NormalizeFormat(f))
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

	m.choices = make([]string, len(pairs))
	m.choiceIcons = make([]string, len(pairs))
	for i, p := range pairs {
		m.choices[i] = strings.ToUpper(p.To)
		m.choiceIcons[i] = ui.PrintFormatCategory(p.To)
	}
	m.cursor = 0

	if isBatch {
		m.state = stateBatchSelectTargetFormat
	} else {
		m.state = stateSelectTargetFormat
	}

	return m
}

func (m interactiveModel) goToFileSelect() interactiveModel {
	m.state = stateSelectFile
	m.cursor = 0

	// Mevcut dizindeki dosyaları tara
	cwd, _ := os.Getwd()
	ext := "." + m.sourceFormat
	m.files = []string{}

	entries, err := os.ReadDir(cwd)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.ToLower(filepath.Ext(e.Name())) == ext {
				m.files = append(m.files, filepath.Join(cwd, e.Name()))
			}
		}
	}

	sort.Strings(m.files)
	return m
}

func (m interactiveModel) doConvert() tea.Cmd {
	return func() tea.Msg {
		start := time.Now()

		conv, err := converter.FindConverter(m.sourceFormat, m.targetFormat)
		if err != nil {
			return convertDoneMsg{err: err, duration: time.Since(start)}
		}

		outputPath := converter.BuildOutputPath(m.selectedFile, outputDir, m.targetFormat, "")
		opts := converter.Options{Quality: 0, Verbose: false}

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

		// Dosyaları topla
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

			outputPath := converter.BuildOutputPath(f, outputDir, m.targetFormat, "")
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

func getUniqueSourceFormats() []string {
	pairs := converter.GetAllConversions()
	formatSet := make(map[string]bool)
	for _, p := range pairs {
		formatSet[p.From] = true
	}

	var formats []string
	for f := range formatSet {
		formats = append(formats, strings.ToUpper(f))
	}
	sort.Strings(formats)
	return formats
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
// ========================================

// RunInteractive interaktif TUI modunu başlatır
func RunInteractive() error {
	p := tea.NewProgram(newInteractiveModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
