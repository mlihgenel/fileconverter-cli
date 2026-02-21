package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mlihgenel/fileconverter-cli/internal/converter"
)

var videoTrimInputFormats = []string{"mp4", "mov", "mkv", "avi", "webm", "m4v", "wmv", "flv"}

const (
	trimRangeDuration = "duration"
	trimRangeEnd      = "end"
)

func (m interactiveModel) goToVideoTrimBrowser() interactiveModel {
	m.flowIsBatch = false
	m.flowResizeOnly = false
	m.flowIsWatch = false
	m.flowVideoTrim = true
	m.resetResizeState()
	m.sourceFormat = ""
	m.targetFormat = ""
	m.selectedFile = ""
	m.selectedCategory = videoCategoryIndex()
	m.trimStartInput = "0"
	m.trimDurationInput = "10"
	m.trimEndInput = ""
	m.trimRangeType = trimRangeDuration
	m.trimMode = trimModeClip
	m.trimCodec = "auto"
	m.trimCodecNote = ""
	m.trimValidationErr = ""
	m.trimPreviewPlan = nil
	m.state = stateFileBrowser
	m.cursor = 0
	if strings.TrimSpace(m.browserDir) == "" {
		m.browserDir = m.defaultOutput
	}
	m.loadBrowserItems()
	return m
}

func videoCategoryIndex() int {
	for i, cat := range categories {
		if cat.Name == "Video Dosyaları" {
			return i
		}
	}
	return 0
}

func isVideoTrimSourceFile(name string) bool {
	for _, format := range videoTrimInputFormats {
		if converter.HasFormatExtension(name, format) {
			return true
		}
	}
	return false
}

type videoTrimExecution struct {
	Input         string
	Output        string
	Mode          string
	Codec         string
	CodecNote     string
	Quality       int
	TargetFormat  string
	StartValue    string
	EndValue      string
	DurationValue string
	Skip          bool
	Plan          videoTrimPlan
}

func (m interactiveModel) buildVideoTrimExecution() (videoTrimExecution, error) {
	execPlan := videoTrimExecution{}
	inputFile := strings.TrimSpace(m.selectedFile)
	if inputFile == "" {
		return execPlan, fmt.Errorf("trim için video seçilmedi")
	}

	mode := normalizeTrimMode(m.trimMode)
	if mode == "" {
		mode = trimModeClip
	}
	rangeType := m.trimRangeType
	if rangeType != trimRangeEnd {
		rangeType = trimRangeDuration
	}
	requestedCodec := normalizeTrimCodec(m.trimCodec)
	if requestedCodec == "" {
		requestedCodec = "auto"
	}

	startValue, err := normalizeVideoTrimTime(m.trimStartInput, true)
	if err != nil {
		return execPlan, fmt.Errorf("geçersiz başlangıç değeri")
	}

	endValue := ""
	durationValue := ""
	if rangeType == trimRangeEnd {
		endValue, err = normalizeVideoTrimTime(m.trimEndInput, true)
		if err != nil {
			return execPlan, fmt.Errorf("geçersiz bitiş değeri")
		}
	} else {
		durationValue, err = normalizeVideoTrimTime(m.trimDurationInput, false)
		if err != nil {
			return execPlan, fmt.Errorf("geçersiz süre değeri")
		}
	}
	startValue, endValue, durationValue, _, _, err = resolveTrimRange(startValue, endValue, durationValue, mode)
	if err != nil {
		return execPlan, err
	}

	format := converter.NormalizeFormat(m.targetFormat)
	if format == "" {
		format = converter.DetectFormat(inputFile)
	}
	if format == "" {
		return execPlan, fmt.Errorf("hedef format belirlenemedi")
	}
	effectiveCodec, codecNote, err := resolveEffectiveTrimCodec(inputFile, format, requestedCodec)
	if err != nil {
		return execPlan, err
	}

	outputBaseDir := strings.TrimSpace(m.defaultOutput)
	if outputBaseDir == "" {
		outputBaseDir = filepath.Dir(inputFile)
	}

	baseName := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
	suffix := "_trim"
	if mode == trimModeRemove {
		suffix = "_cut"
	}
	outputPath := filepath.Join(outputBaseDir, fmt.Sprintf("%s%s.%s", baseName, suffix, format))

	conflictMode := converter.NormalizeConflictPolicy(m.defaultOnConflict)
	if conflictMode == "" {
		conflictMode = converter.ConflictVersioned
	}
	resolvedOutput, skip, err := converter.ResolveOutputPathConflict(outputPath, conflictMode)
	if err != nil {
		return execPlan, err
	}

	plan, err := buildVideoTrimPlan(
		inputFile,
		resolvedOutput,
		mode,
		startValue,
		endValue,
		durationValue,
		nil,
		effectiveCodec,
		m.defaultQuality,
		converter.MetadataAuto,
		conflictMode,
		skip,
		codecNote,
	)
	if err != nil {
		return execPlan, err
	}

	execPlan = videoTrimExecution{
		Input:         inputFile,
		Output:        resolvedOutput,
		Mode:          mode,
		Codec:         effectiveCodec,
		CodecNote:     codecNote,
		Quality:       m.defaultQuality,
		TargetFormat:  format,
		StartValue:    startValue,
		EndValue:      endValue,
		DurationValue: durationValue,
		Skip:          skip,
		Plan:          plan,
	}
	return execPlan, nil
}

func (m interactiveModel) doVideoTrim() tea.Cmd {
	return func() tea.Msg {
		started := time.Now()
		execution, err := m.buildVideoTrimExecution()
		if err != nil {
			return convertDoneMsg{err: err, duration: time.Since(started)}
		}
		if execution.Skip {
			return convertDoneMsg{
				err:      nil,
				duration: time.Since(started),
				output:   fmt.Sprintf("Atlandı (çakışma): %s", execution.Output),
			}
		}

		if err := os.MkdirAll(filepath.Dir(execution.Output), 0755); err != nil {
			return convertDoneMsg{err: err, duration: time.Since(started)}
		}

		if execution.Mode == trimModeRemove {
			err = runTrimRemoveFFmpeg(
				execution.Input,
				execution.Output,
				execution.StartValue,
				execution.EndValue,
				execution.DurationValue,
				execution.TargetFormat,
				execution.Codec,
				execution.Quality,
				converter.MetadataAuto,
				false,
			)
		} else {
			err = runTrimFFmpeg(
				execution.Input,
				execution.Output,
				execution.StartValue,
				execution.EndValue,
				execution.DurationValue,
				execution.TargetFormat,
				execution.Codec,
				execution.Quality,
				converter.MetadataAuto,
				false,
			)
		}
		return convertDoneMsg{
			err:      err,
			duration: time.Since(started),
			output:   execution.Output,
		}
	}
}

func (m interactiveModel) isVideoTrimTextInputState() bool {
	switch m.state {
	case stateVideoTrimStart, stateVideoTrimDuration:
		return true
	default:
		return false
	}
}

func (m *interactiveModel) appendVideoTrimInput(token string) bool {
	field := m.currentVideoTrimInputField()
	if field == nil {
		return false
	}

	r := []rune(token)
	if len(r) != 1 {
		return false
	}

	ch := r[0]
	if ch >= '0' && ch <= '9' {
		*field += string(ch)
		return true
	}
	if ch == ':' {
		*field += string(ch)
		return true
	}
	if ch == '.' || ch == ',' {
		*field += "."
		return true
	}
	return false
}

func (m *interactiveModel) popVideoTrimInput() {
	field := m.currentVideoTrimInputField()
	if field == nil || *field == "" {
		return
	}
	runes := []rune(*field)
	*field = string(runes[:len(runes)-1])
}

func (m *interactiveModel) currentVideoTrimInputField() *string {
	switch m.state {
	case stateVideoTrimStart:
		return &m.trimStartInput
	case stateVideoTrimDuration:
		if m.trimRangeType == trimRangeEnd {
			return &m.trimEndInput
		}
		return &m.trimDurationInput
	default:
		return nil
	}
}

func (m interactiveModel) viewVideoTrimNumericInput(title string, value string, hint string) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(fmt.Sprintf(" ◆ %s ", title)))
	b.WriteString("\n\n")

	if m.selectedFile != "" {
		b.WriteString(infoStyle.Render(fmt.Sprintf("  Dosya: %s", filepath.Base(m.selectedFile))))
		b.WriteString("\n\n")
	}
	if m.trimMode == trimModeRemove {
		b.WriteString(dimStyle.Render("  Bu işlem seçilen aralığı siler, kalan parçaları birleştirip yeni dosya üretir."))
	} else {
		b.WriteString(dimStyle.Render("  Bu işlem seçtiğiniz aralığı yeni klip dosyası olarak çıkarır, orijinali silmez."))
	}
	b.WriteString("\n\n")

	cursor := " "
	if m.showCursor {
		cursor = "▌"
	}

	b.WriteString(pathStyle.Render(fmt.Sprintf("  > %s%s", value, cursor)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  " + hint))
	b.WriteString("\n")

	if m.trimValidationErr != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("  Hata: " + m.trimValidationErr))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Sayı/zaman gir  •  Backspace Sil  •  Enter Devam  •  Esc Geri"))
	b.WriteString("\n")
	return b.String()
}

func (m interactiveModel) viewVideoTrimCodecSelect() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(fmt.Sprintf(" ◆ Video %s — Codec Modu ", m.videoTrimOperationLabel())))
	b.WriteString("\n\n")

	if m.selectedFile != "" {
		b.WriteString(infoStyle.Render(fmt.Sprintf("  Dosya: %s", filepath.Base(m.selectedFile))))
		b.WriteString("\n")
	}
	if m.trimRangeType == trimRangeEnd {
		b.WriteString(infoStyle.Render(fmt.Sprintf("  Başlangıç: %s   Bitiş: %s", m.trimStartInput, m.trimEndInput)))
	} else {
		b.WriteString(infoStyle.Render(fmt.Sprintf("  Başlangıç: %s   Süre: %s", m.trimStartInput, m.trimDurationInput)))
	}
	b.WriteString("\n\n")

	choices := m.choices
	icons := m.choiceIcons
	descs := m.choiceDescs
	if len(choices) == 0 {
		choices = []string{"Auto (önerilen)", "Copy (hızlı)", "Re-encode (uyumlu)"}
		icons = []string{"🧠", "⚡", "🎞️"}
		descs = []string{
			"Hedef formata göre copy/reencode kararını otomatik verir",
			"Seçilen aralığı hızlıca klip olarak çıkarır, kaliteyi korur",
			"Seçilen aralığı yeniden encode ederek daha uyumlu klip üretir",
		}
	}

	for i, choice := range choices {
		icon := ""
		if i < len(icons) {
			icon = icons[i]
		}
		label := menuLine(icon, choice)
		if i == m.cursor {
			b.WriteString(selectedItemStyle.Render("▸ " + label))
			b.WriteString("\n")
			if i < len(descs) && descs[i] != "" {
				b.WriteString(lipgloss.NewStyle().PaddingLeft(7).Foreground(dimTextColor).Italic(true).Render(descs[i]))
				b.WriteString("\n")
			}
		} else {
			b.WriteString(normalItemStyle.Render("  " + label))
			b.WriteString("\n")
		}
	}

	if m.trimValidationErr != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("  Hata: " + m.trimValidationErr))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Onayla  •  Esc Geri"))
	b.WriteString("\n")
	return b.String()
}

func (m interactiveModel) viewVideoTrimPreview() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(fmt.Sprintf(" ◆ Video %s — Plan Ön İzleme ", m.videoTrimOperationLabel())))
	b.WriteString("\n\n")

	if m.selectedFile != "" {
		b.WriteString(infoStyle.Render(fmt.Sprintf("  Dosya: %s", filepath.Base(m.selectedFile))))
		b.WriteString("\n")
	}

	plan := m.trimPreviewPlan
	if plan == nil {
		b.WriteString(errorStyle.Render("  Plan oluşturulamadı. Lütfen bir önceki adıma dönün."))
		b.WriteString("\n\n")
	} else {
		b.WriteString(infoStyle.Render(fmt.Sprintf("  Çıktı: %s", shortenPath(plan.Output))))
		b.WriteString("\n")
		b.WriteString(infoStyle.Render(fmt.Sprintf("  Codec: %s", strings.ToUpper(plan.Codec))))
		b.WriteString("\n")
		if strings.TrimSpace(plan.CodecNote) != "" {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  Not: %s", plan.CodecNote)))
			b.WriteString("\n")
		}
		if plan.HasSourceDuration {
			b.WriteString(infoStyle.Render(fmt.Sprintf("  Kaynak Süre: %s", formatTrimSecondsHuman(plan.SourceDurationSec))))
			b.WriteString("\n")
		}
		if plan.WouldSkip {
			b.WriteString(errorStyle.Render("  Not: on-conflict=skip nedeniyle bu işlem atlanacak."))
			b.WriteString("\n")
		}

		if plan.Mode == trimModeClip {
			endLabel := "dosya sonu"
			if plan.ClipHasEnd {
				endLabel = formatTrimSecondsHuman(plan.ClipEndSec)
			}
			b.WriteString(infoStyle.Render(fmt.Sprintf("  Klip Aralığı: %s -> %s", formatTrimSecondsHuman(plan.ClipStartSec), endLabel)))
			b.WriteString("\n")
			if plan.ClipHasEnd {
				b.WriteString(infoStyle.Render(fmt.Sprintf("  Tahmini Klip Süresi: %s", formatTrimSecondsHuman(plan.ClipEndSec-plan.ClipStartSec))))
				b.WriteString("\n")
			}
		} else {
			b.WriteString(infoStyle.Render(fmt.Sprintf("  Silinecek Aralıklar: %d", len(plan.RemoveRanges))))
			b.WriteString("\n")
			for i, r := range plan.RemoveRanges {
				b.WriteString(dimStyle.Render(fmt.Sprintf(
					"    %d) %s -> %s (%s)",
					i+1,
					formatTrimSecondsHuman(r.Start),
					formatTrimSecondsHuman(r.End),
					formatTrimSecondsHuman(r.End-r.Start),
				)))
				b.WriteString("\n")
			}
			b.WriteString(infoStyle.Render(fmt.Sprintf("  Korunacak Segmentler: %d", len(plan.KeepSegments))))
			b.WriteString("\n")
			for i, s := range plan.KeepSegments {
				endLabel := "dosya sonu"
				lengthLabel := "bilinmiyor"
				if s.HasEnd {
					endLabel = formatTrimSecondsHuman(s.End)
					lengthLabel = formatTrimSecondsHuman(s.End - s.Start)
				}
				b.WriteString(dimStyle.Render(fmt.Sprintf(
					"    %d) %s -> %s (%s)",
					i+1,
					formatTrimSecondsHuman(s.Start),
					endLabel,
					lengthLabel,
				)))
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")
	for i, choice := range m.choices {
		icon := ""
		if i < len(m.choiceIcons) {
			icon = m.choiceIcons[i]
		}
		label := menuLine(icon, choice)
		if i == m.cursor {
			b.WriteString(selectedItemStyle.Render("▸ " + label))
			b.WriteString("\n")
			if i < len(m.choiceDescs) && m.choiceDescs[i] != "" {
				b.WriteString(lipgloss.NewStyle().PaddingLeft(7).Foreground(dimTextColor).Italic(true).Render(m.choiceDescs[i]))
				b.WriteString("\n")
			}
		} else {
			b.WriteString(normalItemStyle.Render("  " + label))
			b.WriteString("\n")
		}
	}

	if m.trimValidationErr != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("  Hata: " + m.trimValidationErr))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç  •  Esc Geri"))
	b.WriteString("\n")
	return b.String()
}

func (m interactiveModel) viewVideoTrimModeSelect() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" ◆ Video Düzenleme Modu Seçin "))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		icon := ""
		if i < len(m.choiceIcons) {
			icon = m.choiceIcons[i]
		}
		label := menuLine(icon, choice)
		if i == m.cursor {
			b.WriteString(selectedItemStyle.Render("▸ " + label))
			b.WriteString("\n")
			if i < len(m.choiceDescs) && m.choiceDescs[i] != "" {
				b.WriteString(lipgloss.NewStyle().PaddingLeft(7).Foreground(dimTextColor).Italic(true).Render(m.choiceDescs[i]))
				b.WriteString("\n")
			}
		} else {
			b.WriteString(normalItemStyle.Render("  " + label))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç  •  Esc Geri"))
	b.WriteString("\n")
	return b.String()
}

func (m interactiveModel) viewVideoTrimRangeTypeSelect() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" ◆ Zaman Aralığı Tipi Seçin "))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		icon := ""
		if i < len(m.choiceIcons) {
			icon = m.choiceIcons[i]
		}
		label := menuLine(icon, choice)
		if i == m.cursor {
			b.WriteString(selectedItemStyle.Render("▸ " + label))
			b.WriteString("\n")
			if i < len(m.choiceDescs) && m.choiceDescs[i] != "" {
				b.WriteString(lipgloss.NewStyle().PaddingLeft(7).Foreground(dimTextColor).Italic(true).Render(m.choiceDescs[i]))
				b.WriteString("\n")
			}
		} else {
			b.WriteString(normalItemStyle.Render("  " + label))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç  •  Esc Geri"))
	b.WriteString("\n")
	return b.String()
}

func (m interactiveModel) videoTrimOperationLabel() string {
	if m.trimMode == trimModeRemove {
		return "Aralığı Sil"
	}
	return "Klip Çıkarma"
}

func suggestVideoTrimEndFromStart(start string) string {
	startSec, err := parseVideoTrimToSeconds(strings.TrimSpace(start))
	if err != nil {
		return "10"
	}
	return formatSecondsForFFmpeg(startSec + 10)
}
