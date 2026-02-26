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

func (m interactiveModel) goToExtractAudioBrowser() interactiveModel {
	m.flowIsBatch = false
	m.flowResizeOnly = false
	m.flowIsWatch = false
	m.flowVideoTrim = false
	m.flowExtractAudio = true
	m.flowSnapshot = false
	m.flowMerge = false
	m.flowAudioNormalize = false
	m.resetResizeState()
	m.sourceFormat = ""
	m.targetFormat = ""
	m.selectedFile = ""
	m.selectedCategory = videoCategoryIndex()

	m.extractAudioQualityInput = "0"
	m.extractAudioCopyMode = false

	m.state = stateFileBrowser
	m.cursor = 0
	if strings.TrimSpace(m.browserDir) == "" {
		m.browserDir = m.defaultOutput
	}
	m.loadBrowserItems()
	return m
}

func (m interactiveModel) doExtractAudio() tea.Cmd {
	return func() tea.Msg {
		started := time.Now()

		inputFile := strings.TrimSpace(m.selectedFile)
		if inputFile == "" {
			return convertDoneMsg{err: fmt.Errorf("ses çıkarma için video seçilmedi"), duration: time.Since(started)}
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
		baseOutput := filepath.Join(outputBaseDir, fmt.Sprintf("%s_audio.%s", baseName, targetFormat))

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

		quality := m.defaultQuality
		if m.extractAudioQualityInput != "0" && m.extractAudioQualityInput != "" {
			fmt.Sscanf(m.extractAudioQualityInput, "%d", &quality)
		}

		err = runExtractAudioFFmpeg(inputFile, resolvedOutput, targetFormat, quality, m.extractAudioCopyMode, converter.MetadataAuto, false)
		return convertDoneMsg{
			err:      err,
			duration: time.Since(started),
			output:   resolvedOutput,
		}
	}
}

func (m interactiveModel) viewExtractAudioTarget() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" Ses Çıkarma: Hedef Format "))
	b.WriteString("\n\n")

	b.WriteString(breadcrumbStyle.Render(fmt.Sprintf("  Seçilen Video: %s", lipgloss.NewStyle().Bold(true).Foreground(accentColor).Render(filepath.Base(m.selectedFile)))))
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

func (m interactiveModel) viewExtractAudioQuality() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" Ses Çıkarma: Kalite (Kbits/s) "))
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

		if i < len(m.choiceDescs) && m.choiceDescs[i] != "" {
			b.WriteString(descStyle.Render(fmt.Sprintf("      %s", m.choiceDescs[i])))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç  •  Esc Geri"))
	b.WriteString("\n")
	return b.String()
}

func (m interactiveModel) viewExtractAudioCopy() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" Ses Çıkarma: İşlem Modu "))
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

		if i < len(m.choiceDescs) && m.choiceDescs[i] != "" {
			b.WriteString(descStyle.Render(fmt.Sprintf("      %s", m.choiceDescs[i])))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç  •  Esc Geri"))
	b.WriteString("\n")
	return b.String()
}
