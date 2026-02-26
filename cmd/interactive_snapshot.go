package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mlihgenel/fileconverter-cli/internal/converter"
)

func (m interactiveModel) goToSnapshotBrowser() interactiveModel {
	m.flowIsBatch = false
	m.flowResizeOnly = false
	m.flowIsWatch = false
	m.flowVideoTrim = false
	m.flowExtractAudio = false
	m.flowSnapshot = true
	m.flowMerge = false
	m.flowAudioNormalize = false
	m.resetResizeState()
	m.sourceFormat = ""
	m.targetFormat = ""
	m.selectedFile = ""
	m.selectedCategory = videoCategoryIndex()

	m.snapshotTimeInput = "00:00:01"
	m.snapshotQualityInput = "0"

	m.state = stateFileBrowser
	m.cursor = 0
	if strings.TrimSpace(m.browserDir) == "" {
		m.browserDir = m.defaultOutput
	}
	m.loadBrowserItems()
	return m
}

func (m interactiveModel) doSnapshot() tea.Cmd {
	return func() tea.Msg {
		started := time.Now()

		inputFile := strings.TrimSpace(m.selectedFile)
		if inputFile == "" {
			return convertDoneMsg{err: fmt.Errorf("kare yakalama için video seçilmedi"), duration: time.Since(started)}
		}

		targetFormat := converter.NormalizeFormat(m.targetFormat)
		if targetFormat == "" {
			return convertDoneMsg{err: fmt.Errorf("hedef format belirlenemedi"), duration: time.Since(started)}
		}

		outputBaseDir := strings.TrimSpace(m.defaultOutput)
		if outputBaseDir == "" {
			outputBaseDir = filepath.Dir(inputFile)
		}

		baseName := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
		baseOutput := filepath.Join(outputBaseDir, fmt.Sprintf("%s_snapshot.%s", baseName, targetFormat))

		conflictMode := converter.NormalizeConflictPolicy(m.defaultOnConflict)
		if conflictMode == "" {
			conflictMode = converter.ConflictVersioned
		}

		resolvedOutput, skip, err := converter.ResolveOutputPathConflict(baseOutput, conflictMode)
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

		timeAt := strings.TrimSpace(m.snapshotTimeInput)
		quality := m.defaultQuality
		if m.snapshotQualityInput != "0" && m.snapshotQualityInput != "" {
			fmt.Sscanf(m.snapshotQualityInput, "%d", &quality)
		}

		timeSec, err := resolveSnapshotTime(timeAt, inputFile)
		if err != nil {
			return convertDoneMsg{err: err, duration: time.Since(started)}
		}
		err = runSnapshotFFmpeg(inputFile, resolvedOutput, timeSec, targetFormat, quality, false)
		return convertDoneMsg{
			err:      err,
			duration: time.Since(started),
			output:   resolvedOutput,
		}
	}
}

func (m interactiveModel) viewSnapshotTime() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" Kare Yakalama: Zaman Seçimi "))
	b.WriteString("\n\n")

	if m.selectedFile != "" {
		b.WriteString(infoStyle.Render(fmt.Sprintf("  Video: %s", filepath.Base(m.selectedFile))))
		b.WriteString("\n\n")
	}

	b.WriteString(dimStyle.Render("  Hangi saniyeden kare yakalanacağını girin."))
	b.WriteString("\n\n")

	cursor := " "
	if m.showCursor {
		cursor = "▌"
	}

	b.WriteString(pathStyle.Render(fmt.Sprintf("  > %s%s", m.snapshotTimeInput, cursor)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Örnek: 30, 00:01:30 veya %50 (yüzde hesaplanır)"))
	b.WriteString("\n")

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Yaz ve Enter ile Onayla  •  Esc Geri"))
	b.WriteString("\n")
	return b.String()
}

func (m interactiveModel) viewSnapshotTarget() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" Kare Yakalama: Format Seçimi "))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		icon := ""
		if i < len(m.choiceIcons) {
			icon = m.choiceIcons[i]
		}
		line := menuLine(icon, choice)

		if i == m.cursor {
			b.WriteString(selectedItemStyle.Render(fmt.Sprintf("▸ %s", line)))
		} else {
			b.WriteString(normalItemStyle.Render(fmt.Sprintf("  %s", line)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç  •  Esc Geri"))
	b.WriteString("\n")
	return b.String()
}

func (m interactiveModel) viewSnapshotQuality() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" Kare Yakalama: Kalite Ayarı "))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		icon := ""
		if i < len(m.choiceIcons) {
			icon = m.choiceIcons[i]
		}
		line := menuLine(icon, choice)

		if i == m.cursor {
			b.WriteString(selectedItemStyle.Render(fmt.Sprintf("▸ %s", line)))
		} else {
			b.WriteString(normalItemStyle.Render(fmt.Sprintf("  %s", line)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç  •  Esc Geri"))
	b.WriteString("\n")
	return b.String()
}
