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
	minTimelineGapSec = 0.1
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
	m.trimTimelineStart = 0
	m.trimTimelineEnd = 0
	m.trimTimelineMax = 0
	m.trimTimelineStep = 1
	m.trimTimelineKnown = false
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

type videoTrimOutputPreview struct {
	TargetFormat   string
	BaseOutput     string
	ResolvedOutput string
	ConflictPolicy string
	Skip           bool
}

func (m interactiveModel) resolveVideoTrimOutputPreview(mode string) (videoTrimOutputPreview, error) {
	preview := videoTrimOutputPreview{}
	inputFile := strings.TrimSpace(m.selectedFile)
	if inputFile == "" {
		return preview, fmt.Errorf("trim için video seçilmedi")
	}
	mode = normalizeTrimMode(mode)
	if mode == "" {
		mode = trimModeClip
	}

	targetFormat := converter.NormalizeFormat(m.targetFormat)
	if targetFormat == "" {
		targetFormat = converter.DetectFormat(inputFile)
	}
	if targetFormat == "" {
		return preview, fmt.Errorf("hedef format belirlenemedi")
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
	baseOutput := filepath.Join(outputBaseDir, fmt.Sprintf("%s%s.%s", baseName, suffix, targetFormat))

	conflictMode := converter.NormalizeConflictPolicy(m.defaultOnConflict)
	if conflictMode == "" {
		conflictMode = converter.ConflictVersioned
	}
	resolvedOutput, skip, err := converter.ResolveOutputPathConflict(baseOutput, conflictMode)
	if err != nil {
		return preview, err
	}

	preview = videoTrimOutputPreview{
		TargetFormat:   targetFormat,
		BaseOutput:     baseOutput,
		ResolvedOutput: resolvedOutput,
		ConflictPolicy: conflictMode,
		Skip:           skip,
	}
	return preview, nil
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
	outputPreview, err := m.resolveVideoTrimOutputPreview(mode)
	if err != nil {
		return execPlan, err
	}
	effectiveCodec, codecNote, err := resolveEffectiveTrimCodec(inputFile, outputPreview.TargetFormat, requestedCodec)
	if err != nil {
		return execPlan, err
	}

	plan, err := buildVideoTrimPlan(
		inputFile,
		outputPreview.ResolvedOutput,
		mode,
		startValue,
		endValue,
		durationValue,
		nil,
		effectiveCodec,
		m.defaultQuality,
		converter.MetadataAuto,
		outputPreview.ConflictPolicy,
		outputPreview.Skip,
		codecNote,
	)
	if err != nil {
		return execPlan, err
	}

	execPlan = videoTrimExecution{
		Input:         inputFile,
		Output:        outputPreview.ResolvedOutput,
		Mode:          mode,
		Codec:         effectiveCodec,
		CodecNote:     codecNote,
		Quality:       m.defaultQuality,
		TargetFormat:  outputPreview.TargetFormat,
		StartValue:    startValue,
		EndValue:      endValue,
		DurationValue: durationValue,
		Skip:          outputPreview.Skip,
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

func (m *interactiveModel) prepareVideoTrimTimeline() error {
	if strings.TrimSpace(m.selectedFile) == "" {
		return fmt.Errorf("trim için video seçilmedi")
	}

	startRaw := strings.TrimSpace(m.trimStartInput)
	if startRaw == "" {
		startRaw = "0"
	}
	startSec, err := parseVideoTrimToSeconds(startRaw)
	if err != nil {
		return fmt.Errorf("geçersiz başlangıç değeri")
	}

	endSec := 0.0
	if m.trimRangeType == trimRangeEnd {
		endSec, err = parseVideoTrimToSeconds(strings.TrimSpace(m.trimEndInput))
		if err != nil {
			return fmt.Errorf("geçersiz bitiş değeri")
		}
	} else {
		durationSec, parseErr := parseVideoTrimToSeconds(strings.TrimSpace(m.trimDurationInput))
		if parseErr != nil {
			return fmt.Errorf("geçersiz süre değeri")
		}
		endSec = startSec + durationSec
	}

	totalSec, known := probeMediaDurationSeconds(m.selectedFile)
	if known {
		startSec, endSec, err = clampTrimWindowToDuration(startSec, endSec, totalSec, m.trimMode)
		if err != nil {
			return err
		}
		m.trimTimelineMax = totalSec
	} else {
		m.trimTimelineMax = endSec + 15
		if m.trimTimelineMax < 60 {
			m.trimTimelineMax = 60
		}
	}

	m.trimTimelineKnown = known
	m.trimTimelineStart = startSec
	m.trimTimelineEnd = endSec
	if m.trimTimelineStep <= 0 {
		m.trimTimelineStep = 1
	}
	m.syncVideoTrimTimelineInputs()
	return nil
}

func (m *interactiveModel) adjustVideoTrimTimeline(delta float64) {
	if delta == 0 {
		return
	}

	if m.cursor == 0 {
		nextStart := m.trimTimelineStart + delta
		if nextStart < 0 {
			nextStart = 0
		}
		maxStart := m.trimTimelineEnd - minTimelineGapSec
		if nextStart > maxStart {
			nextStart = maxStart
		}
		if m.trimTimelineKnown && nextStart > m.trimTimelineMax-minTimelineGapSec {
			nextStart = m.trimTimelineMax - minTimelineGapSec
		}
		if nextStart < 0 {
			nextStart = 0
		}
		m.trimTimelineStart = nextStart
	} else {
		nextEnd := m.trimTimelineEnd + delta
		minEnd := m.trimTimelineStart + minTimelineGapSec
		if nextEnd < minEnd {
			nextEnd = minEnd
		}
		if m.trimTimelineKnown && nextEnd > m.trimTimelineMax {
			nextEnd = m.trimTimelineMax
		}
		m.trimTimelineEnd = nextEnd
	}

	if !m.trimTimelineKnown && m.trimTimelineEnd > m.trimTimelineMax-1 {
		m.trimTimelineMax = m.trimTimelineEnd + 10
	}

	m.syncVideoTrimTimelineInputs()
}

func (m *interactiveModel) syncVideoTrimTimelineInputs() {
	m.trimStartInput = formatSecondsForFFmpeg(m.trimTimelineStart)
	if m.trimRangeType == trimRangeEnd {
		m.trimEndInput = formatSecondsForFFmpeg(m.trimTimelineEnd)
		return
	}
	duration := m.trimTimelineEnd - m.trimTimelineStart
	if duration < minTimelineGapSec {
		duration = minTimelineGapSec
	}
	m.trimDurationInput = formatSecondsForFFmpeg(duration)
}

func increaseTimelineStep(current float64) float64 {
	steps := []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60}
	for i, s := range steps {
		if current < s {
			return s
		}
		if current == s && i < len(steps)-1 {
			return steps[i+1]
		}
	}
	return steps[len(steps)-1]
}

func decreaseTimelineStep(current float64) float64 {
	steps := []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60}
	for i := len(steps) - 1; i >= 0; i-- {
		s := steps[i]
		if current > s {
			return s
		}
		if current == s && i > 0 {
			return steps[i-1]
		}
	}
	return steps[0]
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

func (m interactiveModel) viewVideoTrimTimeline() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(fmt.Sprintf(" ◆ Video %s — Timeline Ayarı ", m.videoTrimOperationLabel())))
	b.WriteString("\n\n")

	if m.selectedFile != "" {
		b.WriteString(infoStyle.Render(fmt.Sprintf("  Dosya: %s", filepath.Base(m.selectedFile))))
		b.WriteString("\n")
	}
	if outputPreview, err := m.resolveVideoTrimOutputPreview(m.trimMode); err != nil {
		b.WriteString(errorStyle.Render("  Çıktı önizleme hatası: " + err.Error()))
		b.WriteString("\n")
	} else {
		b.WriteString(infoStyle.Render(fmt.Sprintf("  Çıktı (önizleme): %s", shortenPath(outputPreview.ResolvedOutput))))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Çakışma Politikası: %s", outputPreview.ConflictPolicy)))
		b.WriteString("\n")
		if outputPreview.Skip {
			b.WriteString(errorStyle.Render("  Not: mevcut dosya nedeniyle işlem atlanacak (on-conflict=skip)."))
			b.WriteString("\n")
		} else if outputPreview.ResolvedOutput != outputPreview.BaseOutput {
			b.WriteString(dimStyle.Render("  Not: çakışma nedeniyle versioned çıktı yolu kullanılacak."))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	totalLabel := "bilinmiyor"
	if m.trimTimelineKnown {
		totalLabel = formatTrimSecondsHuman(m.trimTimelineMax)
	}
	b.WriteString(infoStyle.Render(fmt.Sprintf("  Video Süresi: %s", totalLabel)))
	b.WriteString("\n")

	if !m.trimTimelineKnown {
		b.WriteString(dimStyle.Render("  Not: ffprobe süreyi okuyamadı, bar tahmini ölçekte gösteriliyor."))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	barWidth := 64
	if m.width > 0 && m.width < 90 {
		barWidth = 42
	}
	b.WriteString("  ")
	b.WriteString(m.videoTrimTimelineBar(barWidth))
	b.WriteString("\n\n")

	startLabel := formatTrimSecondsHuman(m.trimTimelineStart)
	endLabel := formatTrimSecondsHuman(m.trimTimelineEnd)
	lengthLabel := formatTrimSecondsHuman(m.trimTimelineEnd - m.trimTimelineStart)

	startPrefix := "  "
	endPrefix := "  "
	if m.cursor == 0 {
		startPrefix = "▸ "
	} else {
		endPrefix = "▸ "
	}

	b.WriteString(infoStyle.Render(fmt.Sprintf("%sBaşlangıç: %s", startPrefix, startLabel)))
	b.WriteString("\n")
	if m.trimRangeType == trimRangeEnd {
		b.WriteString(infoStyle.Render(fmt.Sprintf("%sBitiş:     %s", endPrefix, endLabel)))
	} else {
		b.WriteString(infoStyle.Render(fmt.Sprintf("%sBitiş:     %s", endPrefix, endLabel)))
	}
	b.WriteString("\n")
	b.WriteString(infoStyle.Render(fmt.Sprintf("  Aralık Süresi: %s", lengthLabel)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Adım: %.1fs", m.trimTimelineStep)))
	b.WriteString("\n")

	if m.trimValidationErr != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("  Hata: " + m.trimValidationErr))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ←/→ Aralığı kaydır  •  ↑/↓ veya Tab odak değiştir (başlangıç/bitiş)"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  [ ] Adım azalt/artır  •  Enter Devam  •  Esc Geri"))
	b.WriteString("\n")
	return b.String()
}

func (m interactiveModel) videoTrimTimelineBar(width int) string {
	if width < 20 {
		width = 20
	}
	maxSec := m.trimTimelineMax
	if maxSec <= 0 {
		maxSec = m.trimTimelineEnd + 15
	}
	if maxSec <= 0 {
		maxSec = 60
	}

	startPos := int((m.trimTimelineStart / maxSec) * float64(width-1))
	endPos := int((m.trimTimelineEnd / maxSec) * float64(width-1))
	if startPos < 0 {
		startPos = 0
	}
	if startPos > width-1 {
		startPos = width - 1
	}
	if endPos < startPos {
		endPos = startPos
	}
	if endPos > width-1 {
		endPos = width - 1
	}

	runes := make([]rune, width)
	for i := 0; i < width; i++ {
		runes[i] = '─'
	}
	for i := startPos; i <= endPos && i < width; i++ {
		runes[i] = '━'
	}
	runes[startPos] = '◆'
	runes[endPos] = '◆'

	rangeStyle := lipgloss.NewStyle().Foreground(accentColor)
	baseStyle := lipgloss.NewStyle().Foreground(dimTextColor)
	markerStyle := lipgloss.NewStyle().Foreground(warningColor).Bold(true)

	var b strings.Builder
	b.WriteString(baseStyle.Render("["))
	for i, r := range runes {
		ch := string(r)
		switch {
		case i == startPos || i == endPos:
			b.WriteString(markerStyle.Render(ch))
		case i >= startPos && i <= endPos:
			b.WriteString(rangeStyle.Render(ch))
		default:
			b.WriteString(baseStyle.Render(ch))
		}
	}
	b.WriteString(baseStyle.Render("]"))
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
