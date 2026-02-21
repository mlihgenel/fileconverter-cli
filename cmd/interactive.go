package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mlihgenel/fileconverter-cli/internal/batch"
	"github.com/mlihgenel/fileconverter-cli/internal/config"
	"github.com/mlihgenel/fileconverter-cli/internal/converter"
	"github.com/mlihgenel/fileconverter-cli/internal/installer"
	convwatch "github.com/mlihgenel/fileconverter-cli/internal/watch"
)

// ========================================
// Renk Paleti ve Stiller
// ========================================

var (
	// Ana renk paleti
	primaryColor   = lipgloss.Color("#334155") // Sade slate
	secondaryColor = lipgloss.Color("#E2E8F0") // Açık logo tonu
	accentColor    = lipgloss.Color("#10B981") // Yeşil
	warningColor   = lipgloss.Color("#F59E0B") // Sarı
	dangerColor    = lipgloss.Color("#EF4444") // Kırmızı
	textColor      = lipgloss.Color("#E2E8F0") // Açık gri
	dimTextColor   = lipgloss.Color("#94A3B8") // Koyu gri
	bgColor        = lipgloss.Color("#0F172A") // Koyu arka plan

	// Sade ton geçişi
	gradientColors = []lipgloss.Color{
		"#F1F5F9", "#CBD5E1", "#94A3B8", "#64748B", "#94A3B8",
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
	{Name: "Belgeler", Icon: "📄", Desc: "MD, HTML, PDF, DOCX, TXT, ODT, RTF, CSV", Formats: []string{"md", "html", "pdf", "docx", "txt", "odt", "rtf", "csv"}},
	{Name: "Ses Dosyaları", Icon: "🎵", Desc: "MP3, WAV, OGG, FLAC, AAC, M4A, WMA, OPUS, WEBM", Formats: []string{"mp3", "wav", "ogg", "flac", "aac", "m4a", "wma", "opus", "webm"}},
	{Name: "Görseller", Icon: "🖼️ ", Desc: "PNG, JPEG, WEBP, BMP, GIF, TIFF, ICO", Formats: []string{"png", "jpg", "webp", "bmp", "gif", "tif", "ico"}},
	{Name: "Video Dosyaları", Icon: "🎬", Desc: "MP4, MOV, MKV, AVI, WEBM, M4V, WMV, FLV (GIF'e dönüştürme dahil)", Formats: []string{"mp4", "mov", "mkv", "avi", "webm", "m4v", "wmv", "flv"}},
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
	stateWatching
	stateVideoTrimMode
	stateVideoTrimStart
	stateVideoTrimRangeType
	stateVideoTrimDuration
	stateVideoTrimTimeline
	stateVideoTrimCodec
	stateVideoTrimPreview
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
	flowIsWatch    bool
	flowVideoTrim  bool

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
	batchSkipped   int
	batchFailed    int

	// CLI varsayılanları
	defaultQuality    int
	defaultOnConflict string
	defaultRetry      int
	defaultRetryDelay time.Duration
	defaultReport     string
	defaultWorkers    int

	// Watch
	watchRecursive   bool
	watchInterval    time.Duration
	watchSettle      time.Duration
	watchLastPoll    time.Time
	watchProcessing  bool
	watcher          *convwatch.Watcher
	watchTotal       int
	watchSucceeded   int
	watchSkipped     int
	watchFailed      int
	watchLastStatus  string
	watchLastError   string
	watchStartedAt   time.Time
	watchLastBatchAt time.Time

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

	// Video trim
	trimStartInput     string
	trimDurationInput  string
	trimEndInput       string
	trimRangeType      string
	trimMode           string
	trimCodec          string
	trimCodecNote      string
	trimTimelineStart  float64
	trimTimelineEnd    float64
	trimTimelineMax    float64
	trimTimelineStep   float64
	trimTimelineKnown  bool
	trimTimelineCursor float64
	trimSegments       []trimRange
	trimActiveSegment  int
	trimValidationErr  string
	trimPreviewPlan    *videoTrimPlan
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
	skipped   int
	failed    int
	duration  time.Duration
}

type installDoneMsg struct {
	err error
}

type watchStartedMsg struct {
	watcher *convwatch.Watcher
	err     error
}

type watchCycleMsg struct {
	total     int
	succeeded int
	skipped   int
	failed    int
	err       error
}

type tickMsg time.Time

func newInteractiveModel(deps []converter.ExternalTool, firstRun bool) interactiveModel {
	homeDir := getHomeDir()
	defaults := loadInteractiveDefaults()

	initialState := stateMainMenu
	if firstRun {
		initialState = stateWelcomeIntro
	}

	// Varsayılan çıktı dizinini CLI/env/project config'den çöz.
	selectedOutput := strings.TrimSpace(outputDir)
	if selectedOutput == "" {
		selectedOutput = config.GetDefaultOutputDir()
	}
	if selectedOutput == "" {
		selectedOutput = filepath.Join(homeDir, "Desktop")
	}

	return interactiveModel{
		state:  initialState,
		cursor: 0,
		choices: []string{
			"Dosya Dönüştür",
			"Toplu Dönüştür (Batch)",
			"Klasör İzle (Watch)",
			"Video Düzenle (Klip/Sil)",
			"Boyutlandır",
			"Toplu Boyutlandır",
			"Desteklenen Formatlar",
			"Sistem Kontrolü",
			"Ayarlar",
			"Çıkış",
		},
		choiceIcons: []string{"🔄", "📦", "👀", "✂️", "📐", "🗂️", "📋", "🔧", "⚙️", "👋"},
		choiceDescs: []string{
			"Tek bir dosyayı başka formata dönüştür",
			"Dizindeki tüm dosyaları toplu dönüştür",
			"Klasörde yeni dosyaları izleyip otomatik dönüştür",
			"Videodan aralık çıkarır veya aralığı silip kalanları birleştirir",
			"Tek dosya için görsel/video boyutlandırma",
			"Dizindeki dosyalar için toplu boyutlandırma",
			"Desteklenen format ve dönüşüm yollarını gör",
			"Harici araçların (FFmpeg, LibreOffice, Pandoc) durumu",
			"Varsayılan çıktı dizini ve tercihler",
			"Uygulamadan çık",
		},
		browserDir:        selectedOutput,
		defaultOutput:     selectedOutput,
		width:             80,
		height:            24,
		dependencies:      deps,
		isFirstRun:        firstRun,
		showCursor:        true,
		defaultQuality:    defaults.Quality,
		defaultOnConflict: defaults.OnConflict,
		defaultRetry:      defaults.Retry,
		defaultRetryDelay: defaults.RetryDelay,
		defaultReport:     defaults.Report,
		defaultWorkers:    defaults.Workers,
		watchInterval:     2 * time.Second,
		watchSettle:       1500 * time.Millisecond,
		resizeMethod:      "none",
		resizeModeName:    "pad",
		resizeUnit:        "px",
		resizeDPIInput:    "96",
	}
}

type interactiveDefaults struct {
	Quality    int
	OnConflict string
	Retry      int
	RetryDelay time.Duration
	Report     string
	Workers    int
}

func loadInteractiveDefaults() interactiveDefaults {
	d := interactiveDefaults{
		Quality:    0,
		OnConflict: converter.ConflictVersioned,
		Retry:      0,
		RetryDelay: 500 * time.Millisecond,
		Report:     batch.ReportOff,
		Workers:    workers,
	}
	if d.Workers <= 0 {
		d.Workers = runtime.NumCPU()
	}

	if v, ok := readEnvInt(envQuality); ok && v >= 0 {
		d.Quality = v
	} else if activeProjectConfig != nil && activeProjectConfig.Quality > 0 {
		d.Quality = activeProjectConfig.Quality
	}

	if v := strings.TrimSpace(os.Getenv(envConflict)); v != "" {
		d.OnConflict = v
	} else if activeProjectConfig != nil && strings.TrimSpace(activeProjectConfig.OnConflict) != "" {
		d.OnConflict = activeProjectConfig.OnConflict
	}
	if normalized := converter.NormalizeConflictPolicy(d.OnConflict); normalized != "" {
		d.OnConflict = normalized
	} else {
		d.OnConflict = converter.ConflictVersioned
	}

	if v, ok := readEnvInt(envRetry); ok && v >= 0 {
		d.Retry = v
	} else if activeProjectConfig != nil && activeProjectConfig.Retry > 0 {
		d.Retry = activeProjectConfig.Retry
	}

	if v, ok := readEnvDuration(envRetryDelay); ok && v >= 0 {
		d.RetryDelay = v
	} else if activeProjectConfig != nil && activeProjectConfig.RetryDelay > 0 {
		d.RetryDelay = activeProjectConfig.RetryDelay
	}

	if v := strings.TrimSpace(os.Getenv(envReport)); v != "" {
		d.Report = v
	} else if activeProjectConfig != nil && strings.TrimSpace(activeProjectConfig.ReportFormat) != "" {
		d.Report = activeProjectConfig.ReportFormat
	}
	if normalized := batch.NormalizeReportFormat(d.Report); normalized != "" {
		d.Report = normalized
	} else {
		d.Report = batch.ReportOff
	}

	if v, ok := readEnvInt(envWorkers); ok && v > 0 {
		d.Workers = v
	} else if activeProjectConfig != nil && activeProjectConfig.Workers > 0 {
		d.Workers = activeProjectConfig.Workers
	}

	return d
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
		var watchCmd tea.Cmd

		// Spinner animasyonu
		if m.state == stateConverting || m.state == stateBatchConverting || m.state == stateWelcomeInstalling || m.state == stateMissingDepInstalling || (m.state == stateWatching && m.watchProcessing) {
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

		if m.state == stateWatching && m.watcher != nil && !m.watchProcessing {
			now := time.Now()
			if m.watchLastPoll.IsZero() || now.Sub(m.watchLastPoll) >= m.watchInterval {
				m.watchLastPoll = now
				m.watchProcessing = true
				watchCmd = m.doWatchCycle()
			}
		}

		if watchCmd != nil {
			return m, tea.Batch(tickCmd(), watchCmd)
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
		m.batchSkipped = msg.skipped
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
			// Kurulum başarılı — tek dosyada dönüşüme devam et, batch/watch'ta klasör seçimine dön.
			if m.isBatchPending {
				m.isBatchPending = false
				m.pendingConvertCmd = nil
				m.browserDir = m.defaultOutput
				m.loadBrowserItems()
				m.cursor = 0
				m.state = stateBatchBrowser
				return m, nil
			}
			if m.pendingConvertCmd == nil {
				return m.goToMainMenu(), nil
			} else {
				m.state = stateConverting
				return m, m.pendingConvertCmd
			}
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

	case watchStartedMsg:
		m.watchProcessing = false
		if msg.err != nil {
			m.watchLastError = msg.err.Error()
			m.resultErr = true
			m.resultMsg = msg.err.Error()
			m.state = stateConvertDone
			return m, nil
		}
		m.watcher = msg.watcher
		m.watchStartedAt = time.Now()
		m.watchLastStatus = "İzleme aktif."
		m.watchLastError = ""
		return m, nil

	case watchCycleMsg:
		m.watchProcessing = false
		if msg.err != nil {
			m.watchLastError = msg.err.Error()
			m.watchLastStatus = "İzleme hatası oluştu."
			return m, nil
		}
		m.watchLastError = ""
		m.watchTotal += msg.total
		m.watchSucceeded += msg.succeeded
		m.watchSkipped += msg.skipped
		m.watchFailed += msg.failed
		if msg.total > 0 {
			m.watchLastBatchAt = time.Now()
			m.watchLastStatus = fmt.Sprintf("%d dosya işlendi (ok:%d, atla:%d, hata:%d).", msg.total, msg.succeeded, msg.skipped, msg.failed)
		} else {
			m.watchLastStatus = "Yeni dosya bekleniyor..."
		}
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

		if m.isResizeTextInputState() || m.isVideoTrimTextInputState() {
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
				if m.isResizeTextInputState() {
					m.popResizeInput()
				} else {
					m.popVideoTrimInput()
				}
				return m, nil
			default:
				if m.isResizeTextInputState() && m.appendResizeInput(msg.String()) {
					return m, nil
				}
				if m.isVideoTrimTextInputState() && m.appendVideoTrimInput(msg.String()) {
					return m, nil
				}
				return m, nil
			}
		}
		if m.state == stateVideoTrimTimeline {
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
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
				return m, nil
			case "down", "j":
				if m.cursor < 1 {
					m.cursor++
				}
				return m, nil
			case "tab":
				if m.cursor == 0 {
					m.cursor = 1
				} else {
					m.cursor = 0
				}
				return m, nil
			case "left", "h":
				m.adjustVideoTrimTimeline(-m.trimTimelineStep)
				return m, nil
			case "right", "l":
				m.adjustVideoTrimTimeline(m.trimTimelineStep)
				return m, nil
			case "[":
				m.trimTimelineStep = decreaseTimelineStep(m.trimTimelineStep)
				return m, nil
			case "]":
				m.trimTimelineStep = increaseTimelineStep(m.trimTimelineStep)
				return m, nil
			case ",", "<", "shift+left":
				m.moveTimelineCursor(-m.trimTimelineStep)
				return m, nil
			case ".", ">", "shift+right":
				m.moveTimelineCursor(m.trimTimelineStep)
				return m, nil
			case "a":
				if err := m.addRemoveTimelineSegment(); err != nil {
					m.trimValidationErr = err.Error()
				} else {
					m.trimValidationErr = ""
				}
				return m, nil
			case "n":
				m.selectNextRemoveSegment()
				return m, nil
			case "p":
				m.selectPrevRemoveSegment()
				return m, nil
			case "d":
				if err := m.deleteActiveRemoveSegment(); err != nil {
					m.trimValidationErr = err.Error()
				} else {
					m.trimValidationErr = ""
				}
				return m, nil
			case "m":
				if err := m.mergeRemoveTimelineSegments(); err != nil {
					m.trimValidationErr = err.Error()
				} else {
					m.trimValidationErr = ""
				}
				return m, nil
			case "1", "2", "3", "4", "5", "6", "7", "8", "9":
				if err := m.selectRemoveSegmentByKey(msg.String()); err != nil {
					m.trimValidationErr = err.Error()
				} else {
					m.trimValidationErr = ""
				}
				return m, nil
			default:
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
	case stateWatching:
		return 0
	case stateVideoTrimStart, stateVideoTrimDuration:
		return 0
	case stateVideoTrimTimeline:
		return 1
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
	case stateWatching:
		return m.viewWatching()
	case stateVideoTrimMode:
		return m.viewVideoTrimModeSelect()
	case stateVideoTrimStart:
		return m.viewVideoTrimNumericInput(fmt.Sprintf("Video %s — Başlangıç (sn veya hh:mm:ss)", m.videoTrimOperationLabel()), m.trimStartInput, "Örnek: 23 veya 00:00:23")
	case stateVideoTrimRangeType:
		return m.viewVideoTrimRangeTypeSelect()
	case stateVideoTrimDuration:
		if m.trimRangeType == trimRangeEnd {
			return m.viewVideoTrimNumericInput(fmt.Sprintf("Video %s — Bitiş (sn veya hh:mm:ss)", m.videoTrimOperationLabel()), m.trimEndInput, "Örnek: 25 veya 00:00:25")
		}
		return m.viewVideoTrimNumericInput(fmt.Sprintf("Video %s — Süre (sn veya hh:mm:ss)", m.videoTrimOperationLabel()), m.trimDurationInput, "Örnek: 2 veya 00:00:02")
	case stateVideoTrimTimeline:
		return m.viewVideoTrimTimeline()
	case stateVideoTrimCodec:
		return m.viewVideoTrimCodecSelect()
	case stateVideoTrimPreview:
		return m.viewVideoTrimPreview()
	default:
		return ""
	}
}

// ========================================
// Ekranlar
// ========================================

func (m interactiveModel) viewMainMenu() string {
	var b strings.Builder

	// Ana başlık: ortalı, sade ve şık görünüm.
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#CBD5E1"))
	for _, raw := range welcomeArt {
		line := strings.TrimLeft(raw, " ")
		b.WriteString(centerText(titleStyle.Render(line), m.width))
		b.WriteString("\n")
	}

	// Versiyon bilgisi
	versionLine := fmt.Sprintf("             v%s  •  Yerel & Güvenli Dönüştürücü", appVersion)
	version := lipgloss.NewStyle().Foreground(dimTextColor).Italic(true).Render(versionLine)
	b.WriteString(centerText(version, m.width))
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
	crumb := ""
	if m.flowVideoTrim {
		crumb = fmt.Sprintf("  ✂️ Video Düzenle › %s", lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("Video Seç"))
	} else {
		crumb = fmt.Sprintf("  %s %s › %s › %s",
			cat.Icon,
			cat.Name,
			lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render(strings.ToUpper(m.sourceFormat)),
			lipgloss.NewStyle().Bold(true).Foreground(accentColor).Render(strings.ToUpper(m.targetFormat)))
	}
	b.WriteString(breadcrumbStyle.Render(crumb))
	b.WriteString("\n\n")

	b.WriteString(menuTitleStyle.Render(" ◆ Dosya Seçin "))
	b.WriteString("\n")

	// Mevcut dizin
	shortDir := shortenPath(m.browserDir)
	b.WriteString(pathStyle.Render(fmt.Sprintf("  📁 Dizin: %s", shortDir)))
	b.WriteString("\n\n")

	if len(m.browserItems) == 0 {
		if m.flowVideoTrim {
			b.WriteString(errorStyle.Render("  Bu dizinde video dosyası veya klasör bulunamadı!"))
		} else {
			b.WriteString(errorStyle.Render(fmt.Sprintf("  Bu dizinde .%s dosyası veya klasör bulunamadı!", converter.FormatFilterLabel(m.sourceFormat))))
		}
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
			fileIcon := cat.Icon
			if m.flowVideoTrim {
				fileIcon = "🎬"
			}
			if i == m.cursor {
				b.WriteString(selectedFileStyle.Render(fmt.Sprintf("▸ %s %s", fileIcon, item.name)))
			} else {
				b.WriteString(normalItemStyle.Render(fmt.Sprintf("  %s %s", fileIcon, item.name)))
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
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Ayar: kalite=%d, conflict=%s", m.defaultQuality, m.defaultOnConflict)))
	b.WriteString("\n")
	if m.flowVideoTrim {
		b.WriteString(dimStyle.Render("  Not: Video seçince önce işlem modu seçilir (klip çıkar / aralığı sil)"))
		b.WriteString("\n")
	}
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
		if m.flowVideoTrim {
			content += fmt.Sprintf("  Islem: %s\n", m.videoTrimOperationLabel())
			if m.trimRangeType == trimRangeEnd {
				content += fmt.Sprintf("  Aralik: baslangic=%s, bitis=%s\n", m.trimStartInput, m.trimEndInput)
			} else {
				content += fmt.Sprintf("  Aralik: baslangic=%s, sure=%s\n", m.trimStartInput, m.trimDurationInput)
			}
			codecLabel := strings.ToUpper(m.trimCodec)
			if m.trimPreviewPlan != nil && strings.TrimSpace(m.trimPreviewPlan.Codec) != "" {
				codecLabel = strings.ToUpper(m.trimPreviewPlan.Codec)
			}
			content += fmt.Sprintf("  Codec: %s\n", codecLabel)
			if strings.TrimSpace(m.trimCodecNote) != "" {
				content += fmt.Sprintf("  Codec Kararı: %s\n", m.trimCodecNote)
			}
		}
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
	if m.batchSkipped > 0 {
		content += fmt.Sprintf("  Atlanan:   %d dosya\n", m.batchSkipped)
	}
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
			return m.goToCategorySelect(false, false, false), nil
		case 1:
			return m.goToCategorySelect(true, false, false), nil
		case 2:
			return m.goToCategorySelect(true, false, true), nil
		case 3:
			return m.goToVideoTrimBrowser(), nil
		case 4:
			return m.goToCategorySelect(false, true, false), nil
		case 5:
			return m.goToCategorySelect(true, true, false), nil
		case 6:
			m.state = stateFormats
			m.cursor = 0
			return m, nil
		case 7:
			m.state = stateDependencies
			m.cursor = 0
			return m, nil
		case 8:
			// Ayarlar
			m.state = stateSettings
			m.cursor = 0
			return m, nil
		case 9:
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
				// Dosya seç
				m.selectedFile = item.path
				if m.flowVideoTrim {
					if depName, toolName := m.checkRequiredDep(); depName != "" {
						m.missingDepName = depName
						m.missingDepToolName = toolName
						m.pendingConvertCmd = nil
						m.isBatchPending = false
						m.state = stateMissingDep
						m.cursor = 0
						return m, nil
					}
					m.trimStartInput = "0"
					m.trimDurationInput = "10"
					m.trimEndInput = ""
					m.trimRangeType = trimRangeDuration
					m.trimMode = trimModeClip
					m.trimCodec = "auto"
					m.trimCodecNote = ""
					m.trimSegments = nil
					m.trimActiveSegment = 0
					m.trimValidationErr = ""
					m.trimPreviewPlan = nil
					m.state = stateVideoTrimMode
					m.cursor = 0
					m.choices = []string{"Klip Çıkar (seçilen aralık)", "Aralığı Sil + Birleştir (kalanı koru)"}
					m.choiceIcons = []string{"✂️", "🧩"}
					m.choiceDescs = []string{
						"Seçtiğiniz aralığı yeni bir klip olarak üretir, orijinali korur",
						"Seçtiğiniz aralığı videodan çıkarır ve kalan parçaları birleştirir",
					}
					return m, nil
				}
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

	case stateVideoTrimMode:
		if m.cursor == 1 {
			m.trimMode = trimModeRemove
		} else {
			m.trimMode = trimModeClip
		}
		m.trimCodecNote = ""
		m.trimSegments = nil
		m.trimActiveSegment = 0
		m.trimValidationErr = ""
		m.state = stateVideoTrimStart
		m.cursor = 0
		return m, nil

	case stateVideoTrimStart:
		normalized, err := normalizeVideoTrimTime(m.trimStartInput, true)
		if err != nil {
			m.trimValidationErr = "Geçersiz başlangıç değeri"
			return m, nil
		}
		m.trimStartInput = normalized
		m.trimValidationErr = ""
		m.state = stateVideoTrimRangeType
		m.choices = []string{"Süre ile (duration)", "Bitiş zamanı ile (end)"}
		m.choiceIcons = []string{"⏱️", "🏁"}
		m.choiceDescs = []string{
			"Başlangıçtan itibaren ne kadar süre alınacağını/silineceğini seçersiniz",
			"Başlangıç ve bitiş zamanı vererek aralığı net belirlersiniz",
		}
		if m.trimRangeType == trimRangeEnd {
			m.cursor = 1
		} else {
			m.trimRangeType = trimRangeDuration
			m.cursor = 0
		}
		return m, nil

	case stateVideoTrimRangeType:
		if m.cursor == 1 {
			m.trimRangeType = trimRangeEnd
			if strings.TrimSpace(m.trimEndInput) == "" {
				m.trimEndInput = suggestVideoTrimEndFromStart(m.trimStartInput)
			}
		} else {
			m.trimRangeType = trimRangeDuration
			if strings.TrimSpace(m.trimDurationInput) == "" {
				m.trimDurationInput = "10"
			}
		}
		m.trimValidationErr = ""
		m.state = stateVideoTrimDuration
		m.cursor = 0
		return m, nil

	case stateVideoTrimDuration:
		startValue := m.trimStartInput
		endValue := ""
		durationValue := ""
		if m.trimRangeType == trimRangeEnd {
			normalized, err := normalizeVideoTrimTime(m.trimEndInput, true)
			if err != nil {
				m.trimValidationErr = "Geçersiz bitiş değeri"
				return m, nil
			}
			m.trimEndInput = normalized
			endValue = normalized
		} else {
			normalized, err := normalizeVideoTrimTime(m.trimDurationInput, false)
			if err != nil {
				m.trimValidationErr = "Geçersiz süre değeri"
				return m, nil
			}
			m.trimDurationInput = normalized
			durationValue = normalized
		}
		if _, _, _, _, _, err := resolveTrimRange(startValue, endValue, durationValue, m.trimMode); err != nil {
			m.trimValidationErr = err.Error()
			return m, nil
		}
		if err := m.prepareVideoTrimTimeline(); err != nil {
			m.trimValidationErr = err.Error()
			return m, nil
		}
		m.trimCodecNote = ""
		m.trimValidationErr = ""
		m.state = stateVideoTrimTimeline
		m.cursor = 0
		return m, nil

	case stateVideoTrimTimeline:
		m.trimCodecNote = ""
		m.trimValidationErr = ""
		m.state = stateVideoTrimCodec
		m.cursor = 0
		m.choices = []string{"Auto (önerilen)", "Copy (hızlı)", "Re-encode (uyumlu)"}
		m.choiceIcons = []string{"🧠", "⚡", "🎞️"}
		if m.trimMode == trimModeRemove {
			m.choiceDescs = []string{
				"Hedef formata göre copy/reencode kararını otomatik verir",
				"Aralık silme sonrası kalan parçaları hızlıca birleştirir",
				"Aralık silme sonrası videoyu yeniden encode ederek daha uyumlu çıktı üretir",
			}
		} else {
			m.choiceDescs = []string{
				"Hedef formata göre copy/reencode kararını otomatik verir",
				"Seçilen aralığı hızlıca klip olarak çıkarır, kaliteyi korur",
				"Seçilen aralığı yeniden encode ederek daha uyumlu klip üretir",
			}
		}
		return m, nil

	case stateVideoTrimCodec:
		if m.cursor == 0 {
			m.trimCodec = "auto"
		} else if m.cursor == 1 {
			m.trimCodec = "copy"
		} else {
			m.trimCodec = "reencode"
		}
		execution, err := m.buildVideoTrimExecution()
		if err != nil {
			m.trimValidationErr = err.Error()
			return m, nil
		}
		m.trimValidationErr = ""
		m.trimPreviewPlan = &execution.Plan
		m.trimCodecNote = execution.CodecNote
		m.targetFormat = execution.TargetFormat
		m.state = stateVideoTrimPreview
		m.cursor = 0
		m.choices = []string{"Onayla ve Uygula", "Geri Dön ve Düzenle"}
		m.choiceIcons = []string{"✅", "↩️"}
		m.choiceDescs = []string{
			"Planı onaylayıp video düzenleme işlemini başlatır",
			"Codec/zaman ayarlarına geri döner",
		}
		return m, nil

	case stateVideoTrimPreview:
		if m.cursor == 0 {
			m.trimValidationErr = ""
			m.state = stateConverting
			return m, m.doVideoTrim()
		}
		m.state = stateVideoTrimCodec
		m.cursor = 0
		return m, nil

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
		if m.flowIsWatch {
			m.state = stateWatching
			m.watchLastStatus = "İzleme hazırlanıyor..."
			m.watchProcessing = true
			return m, m.startWatch()
		}
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
	m.flowIsWatch = false
	m.flowVideoTrim = false
	m.watcher = nil
	m.watchProcessing = false
	m.watchLastStatus = ""
	m.watchLastError = ""
	m.watchTotal = 0
	m.watchSucceeded = 0
	m.watchSkipped = 0
	m.watchFailed = 0
	m.watchLastPoll = time.Time{}
	m.watchStartedAt = time.Time{}
	m.watchLastBatchAt = time.Time{}
	m.batchSkipped = 0
	m.resetResizeState()
	m.trimStartInput = ""
	m.trimDurationInput = ""
	m.trimEndInput = ""
	m.trimRangeType = ""
	m.trimMode = ""
	m.trimCodec = ""
	m.trimCodecNote = ""
	m.trimTimelineStart = 0
	m.trimTimelineEnd = 0
	m.trimTimelineMax = 0
	m.trimTimelineStep = 1
	m.trimTimelineKnown = false
	m.trimTimelineCursor = 0
	m.trimSegments = nil
	m.trimActiveSegment = 0
	m.trimValidationErr = ""
	m.trimPreviewPlan = nil
	m.choices = []string{
		"Dosya Dönüştür",
		"Toplu Dönüştür (Batch)",
		"Klasör İzle (Watch)",
		"Video Düzenle (Klip/Sil)",
		"Boyutlandır",
		"Toplu Boyutlandır",
		"Desteklenen Formatlar",
		"Sistem Kontrolü",
		"Ayarlar",
		"Çıkış",
	}
	m.choiceIcons = []string{"🔄", "📦", "👀", "✂️", "📐", "🗂️", "📋", "🔧", "⚙️", "👋"}
	m.choiceDescs = []string{
		"Tek bir dosyayı başka formata dönüştür",
		"Dizindeki tüm dosyaları toplu dönüştür",
		"Klasörde yeni dosyaları izleyip otomatik dönüştür",
		"Videodan aralık çıkarır veya aralığı silip kalanları birleştirir",
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
		return m.goToCategorySelect(false, m.flowResizeOnly, false)
	case stateSelectTargetFormat:
		return m.goToSourceFormatSelect(false)
	case stateFileBrowser:
		if m.flowVideoTrim {
			return m.goToMainMenu()
		}
		if m.flowResizeOnly {
			return m.goToResizeConfig(false)
		}
		return m.goToTargetFormatSelect(false)
	case stateBatchSelectCategory:
		return m.goToMainMenu()
	case stateBatchSelectSourceFormat:
		return m.goToCategorySelect(true, m.flowResizeOnly, m.flowIsWatch)
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
	case stateVideoTrimMode:
		m.state = stateFileBrowser
		m.cursor = 0
		m.trimValidationErr = ""
		return m
	case stateVideoTrimStart:
		m.state = stateVideoTrimMode
		m.cursor = 0
		m.trimValidationErr = ""
		return m
	case stateVideoTrimRangeType:
		m.state = stateVideoTrimStart
		m.cursor = 0
		m.trimValidationErr = ""
		return m
	case stateVideoTrimDuration:
		m.state = stateVideoTrimRangeType
		m.cursor = 0
		m.trimValidationErr = ""
		return m
	case stateVideoTrimTimeline:
		m.state = stateVideoTrimDuration
		m.cursor = 0
		m.trimValidationErr = ""
		return m
	case stateVideoTrimCodec:
		m.state = stateVideoTrimTimeline
		m.cursor = 0
		m.trimValidationErr = ""
		return m
	case stateVideoTrimPreview:
		m.state = stateVideoTrimCodec
		m.cursor = 0
		m.trimValidationErr = ""
		return m
	case stateConvertDone, stateBatchDone, stateFormats:
		return m.goToMainMenu()
	case stateSettings:
		return m.goToMainMenu()
	case stateSettingsBrowser:
		m.state = stateSettings
		m.cursor = 0
		return m
	case stateWatching:
		m.state = stateBatchBrowser
		m.cursor = 0
		m.watchProcessing = false
		m.watcher = nil
		m.watchLastStatus = ""
		m.watchLastError = ""
		return m
	case stateMissingDep:
		return m.goToMainMenu()
	default:
		return m.goToMainMenu()
	}
}

func (m interactiveModel) goToCategorySelect(isBatch bool, resizeOnly bool, isWatch bool) interactiveModel {
	m.flowIsBatch = isBatch
	m.flowResizeOnly = resizeOnly
	m.flowIsWatch = isWatch
	m.flowVideoTrim = false
	m.trimEndInput = ""
	m.trimRangeType = ""
	m.trimMode = ""
	m.trimCodecNote = ""
	m.trimTimelineStart = 0
	m.trimTimelineEnd = 0
	m.trimTimelineMax = 0
	m.trimTimelineStep = 1
	m.trimTimelineKnown = false
	m.trimTimelineCursor = 0
	m.trimSegments = nil
	m.trimActiveSegment = 0
	m.trimValidationErr = ""
	m.trimPreviewPlan = nil
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
		} else if m.flowVideoTrim && isVideoTrimSourceFile(e.Name()) {
			files = append(files, browserEntry{
				name:  e.Name(),
				path:  fullPath,
				isDir: false,
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
		resolvedOutput, skip, err := converter.ResolveOutputPathConflict(outputPath, m.defaultOnConflict)
		if err != nil {
			return convertDoneMsg{err: err, duration: time.Since(start)}
		}
		if skip {
			return convertDoneMsg{
				err:      nil,
				duration: time.Since(start),
				output:   fmt.Sprintf("Atlandı (çakışma): %s", resolvedOutput),
			}
		}
		opts := converter.Options{Quality: m.defaultQuality, Verbose: false, Resize: m.resizeSpec}

		// Çıktı dizininin var olduğundan emin ol
		os.MkdirAll(filepath.Dir(resolvedOutput), 0755)

		err = conv.Convert(m.selectedFile, resolvedOutput, opts)
		return convertDoneMsg{
			err:      err,
			duration: time.Since(start),
			output:   resolvedOutput,
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
		skipped := 0
		failed := 0
		total := len(files)
		reserved := make(map[string]struct{}, len(files))

		jobs := make([]batch.Job, 0, len(files))
		for _, f := range files {
			baseOutput := converter.BuildOutputPath(f, m.defaultOutput, m.targetFormat, "")
			resolvedOutput, skipReason, err := resolveBatchOutputPath(baseOutput, m.defaultOnConflict, reserved)
			if err != nil {
				failed++
				continue
			}
			jobs = append(jobs, batch.Job{
				InputPath:  f,
				OutputPath: resolvedOutput,
				From:       m.sourceFormat,
				To:         m.targetFormat,
				SkipReason: skipReason,
				Options: converter.Options{
					Quality: m.defaultQuality,
					Verbose: false,
					Resize:  m.resizeSpec,
				},
			})
		}

		pool := batch.NewPool(m.defaultWorkers)
		pool.SetRetry(m.defaultRetry, m.defaultRetryDelay)
		results := pool.Execute(jobs)
		summary := batch.GetSummary(results, time.Since(start))
		succeeded = summary.Succeeded
		skipped = summary.Skipped
		failed += summary.Failed

		if m.defaultReport != batch.ReportOff {
			reportText, err := batch.RenderReport(m.defaultReport, summary, results, start, time.Now())
			if err == nil && strings.TrimSpace(reportText) != "" {
				reportPath := filepath.Join(m.defaultOutput, fmt.Sprintf("batch-report-%d.%s", time.Now().Unix(), m.defaultReport))
				_ = writeBatchReport(reportPath, reportText)
			}
		}

		return batchDoneMsg{
			total:     total,
			succeeded: succeeded,
			skipped:   skipped,
			failed:    failed,
			duration:  time.Since(start),
		}
	}
}

func (m interactiveModel) startWatch() tea.Cmd {
	sourceDir := m.browserDir
	if strings.TrimSpace(sourceDir) == "" {
		sourceDir = m.defaultOutput
	}

	return func() tea.Msg {
		w := convwatch.NewWatcher(sourceDir, m.sourceFormat, m.watchRecursive, m.watchSettle)
		if err := w.Bootstrap(); err != nil {
			return watchStartedMsg{err: err}
		}
		return watchStartedMsg{watcher: w}
	}
}

func (m interactiveModel) doWatchCycle() tea.Cmd {
	if m.watcher == nil {
		return func() tea.Msg {
			return watchCycleMsg{}
		}
	}

	return func() tea.Msg {
		files, err := m.watcher.Poll(time.Now())
		if err != nil {
			return watchCycleMsg{err: err}
		}
		if len(files) == 0 {
			return watchCycleMsg{}
		}

		jobs := make([]batch.Job, 0, len(files))
		reserved := make(map[string]struct{}, len(files))
		for _, f := range files {
			baseOutput := converter.BuildOutputPath(f, m.defaultOutput, m.targetFormat, "")
			resolvedOutput, skipReason, err := resolveBatchOutputPath(baseOutput, m.defaultOnConflict, reserved)
			if err != nil {
				return watchCycleMsg{err: err}
			}
			jobs = append(jobs, batch.Job{
				InputPath:  f,
				OutputPath: resolvedOutput,
				From:       m.sourceFormat,
				To:         m.targetFormat,
				SkipReason: skipReason,
				Options: converter.Options{
					Quality: m.defaultQuality,
					Verbose: false,
				},
			})
		}

		pool := batch.NewPool(m.defaultWorkers)
		pool.SetRetry(m.defaultRetry, m.defaultRetryDelay)
		results := pool.Execute(jobs)
		summary := batch.GetSummary(results, 0)

		return watchCycleMsg{
			total:     summary.Total,
			succeeded: summary.Succeeded,
			skipped:   summary.Skipped,
			failed:    summary.Failed,
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

func centerText(text string, width int) string {
	if width <= 0 || lipgloss.Width(text) >= width {
		return text
	}
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, text)
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
	if m.flowVideoTrim {
		if !converter.IsFFmpegAvailable() {
			return "FFmpeg", "ffmpeg"
		}
		return "", ""
	}

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

	b.WriteString(lipgloss.NewStyle().Foreground(textColor).Render("  CLI varsayılanları (env/project config):"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  workers: %d", m.defaultWorkers)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  quality: %d", m.defaultQuality)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  on-conflict: %s", m.defaultOnConflict)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  retry: %d (%s)", m.defaultRetry, m.defaultRetryDelay)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  report: %s", m.defaultReport)))
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
	modeLabel := "Toplu"
	if m.flowIsWatch {
		modeLabel = "Watch"
	}
	crumb := fmt.Sprintf("  %s %s › %s -> %s  (%s)",
		cat.Icon,
		cat.Name,
		lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render(strings.ToUpper(m.sourceFormat)),
		lipgloss.NewStyle().Bold(true).Foreground(accentColor).Render(strings.ToUpper(m.targetFormat)),
		modeLabel)
	b.WriteString(breadcrumbStyle.Render(crumb))
	b.WriteString("\n\n")

	b.WriteString(menuTitleStyle.Render(" Kaynak Dizin Secin "))
	b.WriteString("\n")

	// Mevcut dizin
	shortDir := shortenPath(m.browserDir)
	b.WriteString(pathStyle.Render(fmt.Sprintf("  📁 Dizin: %s", shortDir)))
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
			b.WriteString(selectedItemStyle.Render(fmt.Sprintf("▸ 📁 %s/", item.name)))
		} else {
			b.WriteString(normalItemStyle.Render(fmt.Sprintf("  📁 %s/", folderStyle.Render(item.name))))
		}
		b.WriteString("\n")
		dirIdx++
	}

	// "Dönüştür" butonu
	b.WriteString("\n")
	actionLabel := fmt.Sprintf("🚀 Bu dizindeki %d dosyayi donustur", fileCount)
	if m.flowIsWatch {
		actionLabel = fmt.Sprintf("👀 Bu dizini izle (.%s -> .%s)", converter.FormatFilterLabel(m.sourceFormat), m.targetFormat)
	}
	if m.cursor == dirIdx {
		btn := "▸ " + actionLabel
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(accentColor).Render("  " + btn))
	} else {
		btn := "  " + actionLabel
		b.WriteString(dimStyle.Render("  " + btn))
	}
	b.WriteString("\n")

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç/Gir  •  Esc Geri"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  💾 Cikti: %s", shortenPath(m.defaultOutput))))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Ayar: quality=%d, conflict=%s, retry=%d (%s), report=%s",
		m.defaultQuality, m.defaultOnConflict, m.defaultRetry, m.defaultRetryDelay, m.defaultReport)))
	b.WriteString("\n")
	if m.flowIsWatch {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Watch: interval=%s, settle=%s", m.watchInterval, m.watchSettle)))
		b.WriteString("\n")
	}
	if m.resizeSpec != nil {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Boyutlandirma: %s", m.resizeSummary())))
		b.WriteString("\n")
	}

	return b.String()
}

func (m interactiveModel) viewWatching() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" 👀 Watch Modu "))
	b.WriteString("\n\n")

	sourceDir := m.browserDir
	if strings.TrimSpace(sourceDir) == "" {
		sourceDir = m.defaultOutput
	}

	b.WriteString(pathStyle.Render(fmt.Sprintf("  📁 İzlenen dizin: %s", shortenPath(sourceDir))))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Dönüşüm: .%s -> .%s", converter.FormatFilterLabel(m.sourceFormat), m.targetFormat)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Interval: %s  •  Settle: %s", m.watchInterval, m.watchSettle)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Ayar: quality=%d, conflict=%s, retry=%d (%s)",
		m.defaultQuality, m.defaultOnConflict, m.defaultRetry, m.defaultRetryDelay)))
	b.WriteString("\n\n")

	if m.watchLastStatus != "" {
		b.WriteString(infoStyle.Render("  " + m.watchLastStatus))
		b.WriteString("\n")
	}
	if m.watchLastError != "" {
		b.WriteString(errorStyle.Render("  Hata: " + m.watchLastError))
		b.WriteString("\n")
	}
	if !m.watchStartedAt.IsZero() {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Başlangıç: %s", m.watchStartedAt.Format("2006-01-02 15:04:05"))))
		b.WriteString("\n")
	}
	if !m.watchLastBatchAt.IsZero() {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Son işlem: %s", m.watchLastBatchAt.Format("15:04:05"))))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(successStyle.Render(fmt.Sprintf("  Başarılı:  %d", m.watchSucceeded)))
	b.WriteString("\n")
	if m.watchSkipped > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Atlanan:   %d", m.watchSkipped)))
		b.WriteString("\n")
	}
	if m.watchFailed > 0 {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Başarısız: %d", m.watchFailed)))
		b.WriteString("\n")
	}
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Toplam işlenen: %d", m.watchTotal)))
	b.WriteString("\n\n")

	if m.watchProcessing {
		frame := spinnerFrames[m.spinnerIdx]
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("  " + frame + " Tarama devam ediyor..."))
		b.WriteString("\n\n")
	}

	b.WriteString(dimStyle.Render("  Esc: Watch ekranına geri dön  •  q: Ana menü"))
	b.WriteString("\n")

	return b.String()
}

func RunInteractive() error {
	deps := converter.CheckDependencies()
	firstRun := config.IsFirstRun()
	p := tea.NewProgram(newInteractiveModel(deps, firstRun), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
