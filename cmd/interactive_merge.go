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

func (m interactiveModel) doMerge() tea.Cmd {
	return func() tea.Msg {
		started := time.Now()

		if len(m.mergeFiles) < 2 {
			return convertDoneMsg{err: fmt.Errorf("birleştirme için en az 2 video seçilmelidir"), duration: time.Since(started)}
		}

		targetFormat := converter.NormalizeFormat(m.targetFormat)
		if targetFormat == "" || targetFormat == "ayni format" {
			targetFormat = converter.DetectFormat(m.mergeFiles[0])
		}

		outputBaseDir := strings.TrimSpace(m.defaultOutput)
		if outputBaseDir == "" {
			outputBaseDir = filepath.Dir(m.mergeFiles[0])
		}

		baseName := "merged_video"
		baseOutput := filepath.Join(outputBaseDir, fmt.Sprintf("%s.%s", baseName, targetFormat))

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
		if m.mergeQualityInput != "0" && m.mergeQualityInput != "" {
			fmt.Sscanf(m.mergeQualityInput, "%d", &quality)
		}

		canConcatDemux := !m.mergeReencodeMode && checkCodecConsistency(m.mergeFiles)
		if canConcatDemux {
			err = runMergeConcatDemuxer(m.mergeFiles, resolvedOutput, converter.MetadataAuto, false)
		} else {
			err = runMergeReencode(m.mergeFiles, resolvedOutput, targetFormat, quality, converter.MetadataAuto, false)
		}
		return convertDoneMsg{
			err:      err,
			duration: time.Since(started),
			output:   resolvedOutput,
		}
	}
}

func (m interactiveModel) viewMergeBrowser() string {
	var b strings.Builder

	b.WriteString("\n")
	crumb := fmt.Sprintf("  🔗 %s", lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("Video Birleştirme"))
	b.WriteString(breadcrumbStyle.Render(crumb))
	b.WriteString("\n\n")

	b.WriteString(menuTitleStyle.Render(" ◆ Birleştirilecek Videoları Seçin "))
	b.WriteString("\n")

	shortDir := shortenPath(m.browserDir)
	b.WriteString(pathStyle.Render(fmt.Sprintf("  📁 Dizin: %s", shortDir)))
	b.WriteString("\n\n")

	b.WriteString(infoStyle.Render(fmt.Sprintf("  Seçilen: %d video (Enter ile seçiniz)", len(m.mergeFiles))))
	b.WriteString("\n\n")

	maxVisible := m.height - 14
	if maxVisible < 5 {
		maxVisible = 5
	}
	startIdx := 0
	if m.cursor >= maxVisible {
		startIdx = m.cursor - maxVisible + 1
	}
	endIdx := startIdx + maxVisible

	// +1 for the "Start Merge" button
	totalItems := len(m.browserItems) + 1

	if endIdx > totalItems {
		endIdx = totalItems
	}

	for i := startIdx; i < endIdx; i++ {
		if i == len(m.browserItems) {
			// Start merge button
			b.WriteString("\n")
			if i == m.cursor {
				b.WriteString(selectedItemStyle.Render("▸ 🎬 Birleştirmeyi Başlat"))
			} else {
				b.WriteString(normalItemStyle.Render("  🎬 Birleştirmeyi Başlat"))
			}
			b.WriteString("\n")
			continue
		}

		item := m.browserItems[i]

		isSelected := false
		for _, f := range m.mergeFiles {
			if f == item.path {
				isSelected = true
				break
			}
		}

		checkMark := "[ ]"
		if isSelected {
			checkMark = lipgloss.NewStyle().Foreground(accentColor).Render("[x]")
		}
		if item.isDir {
			checkMark = "   "
		}

		if i == m.cursor {
			if item.isDir {
				b.WriteString(selectedItemStyle.Render(fmt.Sprintf("▸ %s 📁 %s/", checkMark, item.name)))
			} else {
				b.WriteString(selectedFileStyle.Render(fmt.Sprintf("▸ %s 📄 %s", checkMark, item.name)))
			}
		} else {
			if item.isDir {
				b.WriteString(normalItemStyle.Render(fmt.Sprintf("  %s 📁 %s/", checkMark, item.name)))
			} else {
				b.WriteString(normalItemStyle.Render(fmt.Sprintf("  %s 📄 %s", checkMark, item.name)))
			}
		}
		b.WriteString("\n")
	}

	if m.trimValidationErr != "" {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Hata: %s", m.trimValidationErr)))
		b.WriteString("\n\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin · Enter ile seçiniz · esc Geri"))
	return b.String()
}

func (m interactiveModel) viewMergeTarget() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" Video Birleştirme: Hedef Format "))
	b.WriteString("\n\n")

	b.WriteString(breadcrumbStyle.Render(fmt.Sprintf("  %d video birleştirilecek", len(m.mergeFiles))))
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

func (m interactiveModel) viewMergeQuality() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" Video Birleştirme: Kalite Ayarı (CRF) "))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("  Sadece yeniden encode durumunda geçerlidir. Re-encode gerekmiyorsa atlanır."))
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

func (m interactiveModel) viewMergeReencode() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" Video Birleştirme: Kodlama Modu "))
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
