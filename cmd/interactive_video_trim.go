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
	m.trimTimelineCursor = 0
	m.trimSegments = nil
	m.trimActiveSegment = 0
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
	RemoveRanges  []trimRange
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

func (m interactiveModel) removeRangesForExecution() ([]trimRange, error) {
	if normalizeTrimMode(m.trimMode) != trimModeRemove {
		return nil, nil
	}

	if len(m.trimSegments) > 0 {
		ranges := make([]trimRange, 0, len(m.trimSegments))
		for _, r := range m.trimSegments {
			if r.End > r.Start+minTimelineGapSec {
				ranges = append(ranges, r)
			}
		}
		if len(ranges) == 0 {
			return nil, fmt.Errorf("remove işlemi için en az bir geçerli aralık gerekli")
		}
		return mergeTrimRanges(ranges), nil
	}

	return resolveRemoveRanges(m.trimStartInput, m.trimEndInput, m.trimDurationInput, nil)
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
	removeRanges, err := m.removeRangesForExecution()
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
		removeRanges,
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
		RemoveRanges:  removeRanges,
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
			if len(execution.RemoveRanges) > 0 {
				err = runTrimRemoveRangesFFmpeg(
					execution.Input,
					execution.Output,
					execution.RemoveRanges,
					execution.TargetFormat,
					execution.Codec,
					execution.Quality,
					converter.MetadataAuto,
					false,
				)
			} else {
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
			}
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
	if m.trimMode == trimModeRemove {
		if err := m.ensureRemoveTimelineSegments(startSec, endSec); err != nil {
			return err
		}
		if len(m.trimSegments) == 0 {
			return fmt.Errorf("remove işlemi için en az bir aralık gerekli")
		}
		if m.trimActiveSegment < 0 || m.trimActiveSegment >= len(m.trimSegments) {
			m.trimActiveSegment = 0
		}
		m.syncTimelineFromActiveRemoveSegment()
		m.centerTimelineCursorOnActiveSegment()
	} else {
		m.trimTimelineStart = startSec
		m.trimTimelineEnd = endSec
	}
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

	if m.trimMode == trimModeRemove && len(m.trimSegments) > 0 {
		m.adjustActiveRemoveTimelineSegment(delta)
		m.syncVideoTrimTimelineInputs()
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

func (m *interactiveModel) ensureRemoveTimelineSegments(startSec float64, endSec float64) error {
	if len(m.trimSegments) == 0 {
		m.trimSegments = []trimRange{{Start: startSec, End: endSec}}
		m.trimActiveSegment = 0
	}

	ranges := make([]trimRange, 0, len(m.trimSegments))
	for _, r := range m.trimSegments {
		if r.End > r.Start+minTimelineGapSec {
			ranges = append(ranges, r)
		}
	}
	if len(ranges) == 0 {
		return fmt.Errorf("remove işlemi için geçerli segment yok")
	}
	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Start < ranges[j].Start
	})
	if m.trimTimelineKnown {
		clamped, err := clampTrimRangesToDuration(ranges, m.trimTimelineMax)
		if err == nil && len(clamped) > 0 {
			ranges = clamped
		}
	}
	m.trimSegments = ranges
	if m.trimActiveSegment < 0 || m.trimActiveSegment >= len(m.trimSegments) {
		m.trimActiveSegment = 0
	}
	return nil
}

func (m *interactiveModel) syncTimelineFromActiveRemoveSegment() {
	if len(m.trimSegments) == 0 {
		return
	}
	if m.trimActiveSegment < 0 || m.trimActiveSegment >= len(m.trimSegments) {
		m.trimActiveSegment = 0
	}
	active := m.trimSegments[m.trimActiveSegment]
	m.trimTimelineStart = active.Start
	m.trimTimelineEnd = active.End
}

func (m *interactiveModel) adjustActiveRemoveTimelineSegment(delta float64) {
	if len(m.trimSegments) == 0 {
		return
	}
	if m.trimActiveSegment < 0 || m.trimActiveSegment >= len(m.trimSegments) {
		m.trimActiveSegment = 0
	}
	m.syncTimelineFromActiveRemoveSegment()

	active := m.trimSegments[m.trimActiveSegment]
	prevEnd := 0.0
	if m.trimActiveSegment > 0 {
		prevEnd = m.trimSegments[m.trimActiveSegment-1].End
	}
	nextStart := 0.0
	hasNext := false
	if m.trimActiveSegment+1 < len(m.trimSegments) {
		nextStart = m.trimSegments[m.trimActiveSegment+1].Start
		hasNext = true
	}

	if m.cursor == 0 {
		next := active.Start + delta
		minStart := 0.0
		if prevEnd > 0 {
			minStart = prevEnd + minTimelineGapSec
		}
		maxStart := active.End - minTimelineGapSec
		if next < minStart {
			next = minStart
		}
		if next > maxStart {
			next = maxStart
		}
		active.Start = next
	} else {
		next := active.End + delta
		minEnd := active.Start + minTimelineGapSec
		maxEnd := next
		if hasNext {
			maxEnd = nextStart - minTimelineGapSec
		} else if m.trimTimelineKnown {
			maxEnd = m.trimTimelineMax
		}
		if maxEnd < minEnd {
			maxEnd = minEnd
		}
		if next < minEnd {
			next = minEnd
		}
		if next > maxEnd {
			next = maxEnd
		}
		active.End = next
	}

	m.trimSegments[m.trimActiveSegment] = active
	m.trimTimelineStart = active.Start
	m.trimTimelineEnd = active.End
	if m.trimTimelineCursor < active.Start {
		m.trimTimelineCursor = active.Start
	} else if m.trimTimelineCursor > active.End {
		m.trimTimelineCursor = active.End
	}
}

func (m *interactiveModel) addRemoveTimelineSegment() error {
	if m.trimMode != trimModeRemove {
		return fmt.Errorf("çoklu segment yalnızca remove modunda kullanılabilir")
	}
	if len(m.trimSegments) == 0 {
		if err := m.ensureRemoveTimelineSegments(m.trimTimelineStart, m.trimTimelineEnd); err != nil {
			return err
		}
	}
	if m.trimActiveSegment < 0 || m.trimActiveSegment >= len(m.trimSegments) {
		m.trimActiveSegment = 0
	}

	base := m.trimSegments[m.trimActiveSegment]
	start := base.End + minTimelineGapSec
	end := start + maxFloat(1, m.trimTimelineStep*4)

	if m.trimActiveSegment+1 < len(m.trimSegments) {
		nextStart := m.trimSegments[m.trimActiveSegment+1].Start - minTimelineGapSec
		if start >= nextStart {
			return fmt.Errorf("yeni segment için boş alan yok")
		}
		if end > nextStart {
			end = nextStart
		}
	}
	if m.trimTimelineKnown && end > m.trimTimelineMax {
		end = m.trimTimelineMax
	}
	if end-start <= minTimelineGapSec {
		return fmt.Errorf("yeni segment için yeterli alan yok")
	}

	insertAt := m.trimActiveSegment + 1
	m.trimSegments = append(m.trimSegments, trimRange{})
	copy(m.trimSegments[insertAt+1:], m.trimSegments[insertAt:])
	m.trimSegments[insertAt] = trimRange{Start: start, End: end}
	m.trimActiveSegment = insertAt
	m.syncTimelineFromActiveRemoveSegment()
	m.centerTimelineCursorOnActiveSegment()
	m.syncVideoTrimTimelineInputs()
	return nil
}

func (m *interactiveModel) selectNextRemoveSegment() {
	if m.trimMode != trimModeRemove || len(m.trimSegments) == 0 {
		return
	}
	m.trimActiveSegment++
	if m.trimActiveSegment >= len(m.trimSegments) {
		m.trimActiveSegment = 0
	}
	m.syncTimelineFromActiveRemoveSegment()
	m.centerTimelineCursorOnActiveSegment()
	m.syncVideoTrimTimelineInputs()
}

func (m *interactiveModel) selectPrevRemoveSegment() {
	if m.trimMode != trimModeRemove || len(m.trimSegments) == 0 {
		return
	}
	m.trimActiveSegment--
	if m.trimActiveSegment < 0 {
		m.trimActiveSegment = len(m.trimSegments) - 1
	}
	m.syncTimelineFromActiveRemoveSegment()
	m.centerTimelineCursorOnActiveSegment()
	m.syncVideoTrimTimelineInputs()
}

func (m *interactiveModel) deleteActiveRemoveSegment() error {
	if m.trimMode != trimModeRemove {
		return fmt.Errorf("çoklu segment yalnızca remove modunda kullanılabilir")
	}
	if len(m.trimSegments) <= 1 {
		return fmt.Errorf("en az bir segment kalmalı")
	}
	if m.trimActiveSegment < 0 || m.trimActiveSegment >= len(m.trimSegments) {
		m.trimActiveSegment = 0
	}
	idx := m.trimActiveSegment
	m.trimSegments = append(m.trimSegments[:idx], m.trimSegments[idx+1:]...)
	if m.trimActiveSegment >= len(m.trimSegments) {
		m.trimActiveSegment = len(m.trimSegments) - 1
	}
	m.syncTimelineFromActiveRemoveSegment()
	m.centerTimelineCursorOnActiveSegment()
	m.syncVideoTrimTimelineInputs()
	return nil
}

func (m *interactiveModel) mergeRemoveTimelineSegments() error {
	if m.trimMode != trimModeRemove {
		return fmt.Errorf("çoklu segment yalnızca remove modunda kullanılabilir")
	}
	if len(m.trimSegments) == 0 {
		return fmt.Errorf("birleştirilecek segment yok")
	}
	activeStart := m.trimSegments[m.trimActiveSegment].Start
	merged := mergeTrimRanges(m.trimSegments)
	if len(merged) == 0 {
		return fmt.Errorf("birleştirilecek geçerli segment yok")
	}
	m.trimSegments = merged
	m.trimActiveSegment = nearestSegmentIndex(activeStart, merged)
	m.syncTimelineFromActiveRemoveSegment()
	m.centerTimelineCursorOnActiveSegment()
	m.syncVideoTrimTimelineInputs()
	return nil
}

func (m *interactiveModel) moveTimelineCursor(delta float64) {
	if m.trimMode != trimModeRemove || len(m.trimSegments) == 0 || delta == 0 {
		return
	}
	maxSec := m.trimTimelineMax
	if maxSec <= 0 {
		maxSec = m.trimTimelineEnd + 15
		if maxSec < 60 {
			maxSec = 60
		}
	}
	if m.trimTimelineCursor <= 0 {
		m.centerTimelineCursorOnActiveSegment()
	}
	next := m.trimTimelineCursor + delta
	if next < 0 {
		next = 0
	}
	if next > maxSec {
		next = maxSec
	}
	m.trimTimelineCursor = next

	selectedIdx := nearestSegmentIndex(m.trimTimelineCursor, m.trimSegments)
	for i, r := range m.trimSegments {
		if m.trimTimelineCursor >= r.Start && m.trimTimelineCursor <= r.End {
			selectedIdx = i
			break
		}
	}
	m.trimActiveSegment = selectedIdx
	m.syncTimelineFromActiveRemoveSegment()
	m.syncVideoTrimTimelineInputs()
}

func (m *interactiveModel) selectRemoveSegmentByKey(key string) error {
	if m.trimMode != trimModeRemove {
		return nil
	}
	if len(m.trimSegments) == 0 {
		return fmt.Errorf("seçilecek segment yok")
	}
	if len(key) != 1 || key[0] < '1' || key[0] > '9' {
		return fmt.Errorf("geçersiz segment kısayolu: %s", key)
	}

	idx := int(key[0] - '1')
	if idx < 0 || idx >= len(m.trimSegments) {
		return fmt.Errorf("%d. segment mevcut değil", idx+1)
	}
	m.trimActiveSegment = idx
	m.syncTimelineFromActiveRemoveSegment()
	m.centerTimelineCursorOnActiveSegment()
	m.syncVideoTrimTimelineInputs()
	return nil
}

func (m *interactiveModel) centerTimelineCursorOnActiveSegment() {
	if len(m.trimSegments) == 0 {
		m.trimTimelineCursor = 0
		return
	}
	if m.trimActiveSegment < 0 || m.trimActiveSegment >= len(m.trimSegments) {
		m.trimActiveSegment = 0
	}
	active := m.trimSegments[m.trimActiveSegment]
	m.trimTimelineCursor = (active.Start + active.End) / 2
}

func nearestSegmentIndex(anchor float64, segments []trimRange) int {
	if len(segments) == 0 {
		return 0
	}
	bestIdx := 0
	bestDist := absFloat(segments[0].Start - anchor)
	for i := 1; i < len(segments); i++ {
		dist := absFloat(segments[i].Start - anchor)
		if dist < bestDist {
			bestDist = dist
			bestIdx = i
		}
	}
	return bestIdx
}

func maxFloat(a float64, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func timelinePosForSecond(sec float64, maxSec float64, width int) int {
	if width <= 1 || maxSec <= 0 {
		return 0
	}
	if sec < 0 {
		sec = 0
	}
	if sec > maxSec {
		sec = maxSec
	}
	pos := int((sec / maxSec) * float64(width-1))
	if pos < 0 {
		return 0
	}
	if pos >= width {
		return width - 1
	}
	return pos
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
	if m.trimMode == trimModeRemove {
		b.WriteString(infoStyle.Render(fmt.Sprintf("  İmleç: %s", formatTrimSecondsHuman(m.trimTimelineCursor))))
		b.WriteString("\n")
	}
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Adım: %.1fs", m.trimTimelineStep)))
	b.WriteString("\n")

	if m.trimMode == trimModeRemove {
		b.WriteString("\n")
		segmentCount := len(m.trimSegments)
		activeLabel := "yok"
		if segmentCount > 0 {
			activeLabel = fmt.Sprintf("%d/%d", m.trimActiveSegment+1, segmentCount)
		}
		b.WriteString(infoStyle.Render(fmt.Sprintf("  Silinecek Segmentler: %d  •  Aktif: %s", segmentCount, activeLabel)))
		b.WriteString("\n")
		visible := segmentCount
		if visible > 6 {
			visible = 6
		}
		for i := 0; i < visible; i++ {
			r := m.trimSegments[i]
			prefix := "   "
			if i == m.trimActiveSegment {
				prefix = " ▸ "
			}
			line := fmt.Sprintf(
				"%s%d) %s -> %s (%s)",
				prefix,
				i+1,
				formatTrimSecondsHuman(r.Start),
				formatTrimSecondsHuman(r.End),
				formatTrimSecondsHuman(r.End-r.Start),
			)
			if i == m.trimActiveSegment {
				b.WriteString(infoStyle.Render(line))
			} else {
				b.WriteString(dimStyle.Render(line))
			}
			b.WriteString("\n")
		}
		if segmentCount > visible {
			b.WriteString(dimStyle.Render(fmt.Sprintf("   ... (%d segment daha)", segmentCount-visible)))
			b.WriteString("\n")
		}
	}

	if m.trimValidationErr != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("  Hata: " + m.trimValidationErr))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ←/→ Aktif aralık sınırını değiştir  •  ↑/↓ veya Tab odak değiştir (başlangıç/bitiş)"))
	b.WriteString("\n")
	if m.trimMode == trimModeRemove {
		b.WriteString(dimStyle.Render("  ,/. İmleç taşı (en yakın segment aktif olur)  •  1-9 Direkt segment seç"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  a Yeni segment  •  n/p Segment gez  •  d Sil  •  m Birleştir"))
		b.WriteString("\n")
	}
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

	const (
		timelineBase = iota
		timelineRange
		timelineActiveRange
		timelineMarker
		timelineCursor
	)

	runes := make([]rune, width)
	styles := make([]int, width)
	for i := 0; i < width; i++ {
		runes[i] = '─'
		styles[i] = timelineBase
	}

	setRange := func(startSec float64, endSec float64, active bool) {
		startPos := timelinePosForSecond(startSec, maxSec, width)
		endPos := timelinePosForSecond(endSec, maxSec, width)
		if endPos < startPos {
			endPos = startPos
		}
		fillStyle := timelineRange
		if active {
			fillStyle = timelineActiveRange
		}
		for i := startPos; i <= endPos && i < width; i++ {
			runes[i] = '━'
			if fillStyle > styles[i] {
				styles[i] = fillStyle
			}
		}
		runes[startPos] = '◆'
		runes[endPos] = '◆'
		styles[startPos] = timelineMarker
		styles[endPos] = timelineMarker
	}

	if m.trimMode == trimModeRemove && len(m.trimSegments) > 0 {
		for i, seg := range m.trimSegments {
			setRange(seg.Start, seg.End, i == m.trimActiveSegment)
		}
	} else {
		setRange(m.trimTimelineStart, m.trimTimelineEnd, true)
	}

	if m.trimMode == trimModeRemove {
		cursorPos := timelinePosForSecond(m.trimTimelineCursor, maxSec, width)
		runes[cursorPos] = '│'
		styles[cursorPos] = timelineCursor
	}

	baseStyle := lipgloss.NewStyle().Foreground(dimTextColor)
	rangeStyle := lipgloss.NewStyle().Foreground(secondaryColor)
	activeStyle := lipgloss.NewStyle().Foreground(accentColor)
	markerStyle := lipgloss.NewStyle().Foreground(warningColor).Bold(true)
	cursorStyle := lipgloss.NewStyle().Foreground(textColor).Bold(true)

	var b strings.Builder
	b.WriteString(baseStyle.Render("["))
	for i, r := range runes {
		ch := string(r)
		switch styles[i] {
		case timelineRange:
			b.WriteString(rangeStyle.Render(ch))
		case timelineActiveRange:
			b.WriteString(activeStyle.Render(ch))
		case timelineMarker:
			b.WriteString(markerStyle.Render(ch))
		case timelineCursor:
			b.WriteString(cursorStyle.Render(ch))
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
