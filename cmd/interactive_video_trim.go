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
	m.trimMode = trimModeClip
	m.trimCodec = "copy"
	m.trimValidationErr = ""
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

func (m interactiveModel) doVideoTrim() tea.Cmd {
	inputFile := m.selectedFile
	startInput := m.trimStartInput
	durationInput := m.trimDurationInput
	mode := m.trimMode
	if normalizeTrimMode(mode) == "" {
		mode = trimModeClip
	}
	codec := m.trimCodec
	quality := m.defaultQuality
	outputBaseDir := m.defaultOutput
	conflictMode := m.defaultOnConflict
	targetFormat := m.targetFormat

	return func() tea.Msg {
		started := time.Now()

		if strings.TrimSpace(inputFile) == "" {
			return convertDoneMsg{err: fmt.Errorf("trim için video seçilmedi"), duration: time.Since(started)}
		}

		startValue, err := normalizeVideoTrimTime(startInput, true)
		if err != nil {
			return convertDoneMsg{err: fmt.Errorf("geçersiz başlangıç değeri"), duration: time.Since(started)}
		}
		durationValue, err := normalizeVideoTrimTime(durationInput, false)
		if err != nil {
			return convertDoneMsg{err: fmt.Errorf("geçersiz süre değeri"), duration: time.Since(started)}
		}
		startValue, _, durationValue, _, _, err = resolveTrimRange(startValue, "", durationValue, mode)
		if err != nil {
			return convertDoneMsg{err: err, duration: time.Since(started)}
		}

		format := converter.NormalizeFormat(targetFormat)
		if format == "" {
			format = converter.DetectFormat(inputFile)
		}
		if format == "" {
			return convertDoneMsg{err: fmt.Errorf("hedef format belirlenemedi"), duration: time.Since(started)}
		}

		if strings.TrimSpace(outputBaseDir) == "" {
			outputBaseDir = filepath.Dir(inputFile)
		}

		baseName := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
		suffix := "_trim"
		if mode == trimModeRemove {
			suffix = "_cut"
		}
		outputPath := filepath.Join(outputBaseDir, fmt.Sprintf("%s%s.%s", baseName, suffix, format))

		resolvedOutput, skip, err := converter.ResolveOutputPathConflict(outputPath, conflictMode)
		if err != nil {
			return convertDoneMsg{err: err, duration: time.Since(started)}
		}
		if skip {
			return convertDoneMsg{
				err:      nil,
				duration: time.Since(started),
				output:   fmt.Sprintf("Atlandı (çakışma): %s", resolvedOutput),
			}
		}

		if err := os.MkdirAll(filepath.Dir(resolvedOutput), 0755); err != nil {
			return convertDoneMsg{err: err, duration: time.Since(started)}
		}

		if mode == trimModeRemove {
			err = runTrimRemoveFFmpeg(inputFile, resolvedOutput, startValue, "", durationValue, codec, quality, converter.MetadataAuto, false)
		} else {
			err = runTrimFFmpeg(inputFile, resolvedOutput, startValue, "", durationValue, codec, quality, converter.MetadataAuto, false)
		}
		return convertDoneMsg{
			err:      err,
			duration: time.Since(started),
			output:   resolvedOutput,
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
	b.WriteString(infoStyle.Render(fmt.Sprintf("  Başlangıç: %s   Süre: %s", m.trimStartInput, m.trimDurationInput)))
	b.WriteString("\n\n")

	choices := m.choices
	icons := m.choiceIcons
	descs := m.choiceDescs
	if len(choices) == 0 {
		choices = []string{"Copy (hızlı)", "Re-encode (uyumlu)"}
		icons = []string{"⚡", "🎞️"}
		descs = []string{
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

func (m interactiveModel) videoTrimOperationLabel() string {
	if m.trimMode == trimModeRemove {
		return "Aralığı Sil"
	}
	return "Klip Çıkarma"
}
