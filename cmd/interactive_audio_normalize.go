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

func (m interactiveModel) goToAudioNormalizeBrowser() interactiveModel {
	m.flowIsBatch = false
	m.flowResizeOnly = false
	m.flowIsWatch = false
	m.flowVideoTrim = false
	m.flowExtractAudio = false
	m.flowSnapshot = false
	m.flowMerge = false
	m.flowAudioNormalize = true
	m.resetResizeState()
	m.sourceFormat = ""
	m.targetFormat = ""
	m.selectedFile = ""

	// Audio category
	for i, cat := range categories {
		if cat.Name == "Ses Dosyaları" {
			m.selectedCategory = i
			break
		}
	}

	m.normalizeLUFSInput = "-14.0"
	m.normalizeTPInput = "-1.0"
	m.normalizeLRAInput = "11.0"

	m.state = stateFileBrowser
	m.cursor = 0
	if strings.TrimSpace(m.browserDir) == "" {
		m.browserDir = m.defaultOutput
	}
	m.loadBrowserItems()
	return m
}

func (m interactiveModel) doAudioNormalize() tea.Cmd {
	return func() tea.Msg {
		started := time.Now()

		inputFile := strings.TrimSpace(m.selectedFile)
		if inputFile == "" {
			return convertDoneMsg{err: fmt.Errorf("ses normalize için dosya seçilmedi"), duration: time.Since(started)}
		}

		targetFormat := converter.NormalizeFormat(m.targetFormat)
		if targetFormat == "" || targetFormat == "ayni format" {
			targetFormat = converter.DetectFormat(inputFile)
		}

		outputBaseDir := strings.TrimSpace(m.defaultOutput)
		if outputBaseDir == "" {
			outputBaseDir = filepath.Dir(inputFile)
		}

		baseName := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
		baseOutput := filepath.Join(outputBaseDir, fmt.Sprintf("%s_norm.%s", baseName, targetFormat))

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

		targetLUFS := -14.0
		targetTP := -1.0
		targetLRA := 11.0
		fmt.Sscanf(m.normalizeLUFSInput, "%f", &targetLUFS)
		fmt.Sscanf(m.normalizeTPInput, "%f", &targetTP)
		fmt.Sscanf(m.normalizeLRAInput, "%f", &targetLRA)

		err = runAudioNormalizeFFmpeg(inputFile, resolvedOutput, targetFormat, targetLUFS, targetTP, targetLRA, converter.MetadataAuto, false)
		return convertDoneMsg{
			err:      err,
			duration: time.Since(started),
			output:   resolvedOutput,
		}
	}
}

func (m interactiveModel) viewAudioNormalizeTarget() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" Ses Normalize: Hedef Format "))
	b.WriteString("\n\n")

	b.WriteString(breadcrumbStyle.Render(fmt.Sprintf("  Seçilen Ses: %s", lipgloss.NewStyle().Bold(true).Foreground(accentColor).Render(filepath.Base(m.selectedFile)))))
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

func (m interactiveModel) viewAudioNormalizeLUFS() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" Ses Normalize: Hedef LUFS (Loudness) "))
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

func (m interactiveModel) viewAudioNormalizeTP() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" Ses Normalize: True Peak (TP) Limiti "))
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

func (m interactiveModel) viewAudioNormalizeLRA() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" Ses Normalize: Loudness Range (LRA) "))
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
