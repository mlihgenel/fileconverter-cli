package cmd

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mlihgenel/fileconverter-cli/internal/converter"
)

type resizeModeOption struct {
	Key   string
	Label string
	Desc  string
}

var resizeModeOptions = []resizeModeOption{
	{Key: "pad", Label: "PAD (Siyah boşluk)", Desc: "Oranı korur, hedef alanı siyah boşlukla tamamlar"},
	{Key: "fit", Label: "FIT (Sığdır)", Desc: "Oranı korur, görüntüyü hedef alanın içine sığdırır"},
	{Key: "fill", Label: "FILL (Kırparak doldur)", Desc: "Oranı korur, taşan alanı merkezden kırpar"},
	{Key: "stretch", Label: "STRETCH (Esnet)", Desc: "Oranı korumaz, hedef boyuta zorla esnetir"},
}

func (m *interactiveModel) resetResizeState() {
	m.resizeIsBatchFlow = false
	m.resizeSpec = nil
	m.resizeMethod = "none"
	m.resizePresetList = nil
	m.resizePresetName = ""
	m.resizeModeName = "pad"
	m.resizeWidthInput = ""
	m.resizeHeightInput = ""
	m.resizeUnit = "px"
	m.resizeDPIInput = "96"
	m.resizeValidationErr = ""
}

func (m interactiveModel) canConfigureResize() bool {
	return converter.IsResizableFormat(m.sourceFormat)
}

func (m interactiveModel) goToResizeConfig(isBatch bool) interactiveModel {
	if m.resizeMethod == "" {
		m.resizeMethod = "none"
	}
	if m.resizeModeName == "" {
		m.resizeModeName = "pad"
	}
	if m.resizeUnit == "" {
		m.resizeUnit = "px"
	}
	if m.resizeDPIInput == "" {
		m.resizeDPIInput = "96"
	}

	m.resizeIsBatchFlow = isBatch
	m.state = stateResizeConfig
	m.cursor = 0
	m.resizeValidationErr = ""

	m.choices = []string{
		"Boyutlandırma kapalı",
		"Hazır ölçü seç (Preset)",
		"Manuel ölçü gir (Elle)",
	}
	m.choiceIcons = []string{"", "", ""}
	m.choiceDescs = []string{
		"Orijinal çözünürlük korunur",
		"Story, square, fullhd gibi hazır ölçüler",
		"Piksel veya santimetre girerek özel ölçü belirle",
	}

	switch m.resizeMethod {
	case "preset":
		m.cursor = 1
	case "manual":
		m.cursor = 2
	default:
		m.cursor = 0
	}

	return m
}

func (m interactiveModel) goToResizePresetSelect() interactiveModel {
	m.state = stateResizePresetSelect
	m.cursor = 0
	m.resizeValidationErr = ""
	m.resizePresetList = converter.ResizePresets()

	m.choices = make([]string, len(m.resizePresetList))
	m.choiceIcons = make([]string, len(m.resizePresetList))
	m.choiceDescs = make([]string, len(m.resizePresetList))
	for i, p := range m.resizePresetList {
		m.choices[i] = fmt.Sprintf("%s (%dx%d)", strings.ToUpper(p.Name), p.Width, p.Height)
		m.choiceIcons[i] = ""
		m.choiceDescs[i] = p.Description
		if p.Name == m.resizePresetName {
			m.cursor = i
		}
	}

	return m
}

func (m interactiveModel) goToResizeManualWidth() interactiveModel {
	m.state = stateResizeManualWidth
	m.cursor = 0
	m.resizeValidationErr = ""
	return m
}

func (m interactiveModel) goToResizeManualHeight() interactiveModel {
	m.state = stateResizeManualHeight
	m.cursor = 0
	m.resizeValidationErr = ""
	return m
}

func (m interactiveModel) goToResizeManualUnitSelect() interactiveModel {
	m.state = stateResizeManualUnit
	m.resizeValidationErr = ""
	m.choices = []string{"Piksel (px)", "Santimetre (cm)"}
	m.choiceIcons = []string{"", ""}
	m.choiceDescs = []string{
		"Doğrudan ekran/video çözünürlüğü girilir",
		"DPI ile piksele çevrilir (örn. baskı iş akışı)",
	}
	if m.resizeUnit == "cm" {
		m.cursor = 1
	} else {
		m.cursor = 0
	}
	return m
}

func (m interactiveModel) goToResizeManualDPI() interactiveModel {
	m.state = stateResizeManualDPI
	m.cursor = 0
	m.resizeValidationErr = ""
	return m
}

func (m interactiveModel) goToResizeModeSelect() interactiveModel {
	m.state = stateResizeModeSelect
	m.resizeValidationErr = ""

	m.choices = make([]string, len(resizeModeOptions))
	m.choiceIcons = make([]string, len(resizeModeOptions))
	m.choiceDescs = make([]string, len(resizeModeOptions))
	m.cursor = 0

	for i, mode := range resizeModeOptions {
		m.choices[i] = mode.Label
		m.choiceIcons[i] = ""
		m.choiceDescs[i] = mode.Desc
		if mode.Key == m.resizeModeName {
			m.cursor = i
		}
	}

	return m
}

func (m interactiveModel) goToBatchBrowserOrDependencyCheck() (tea.Model, tea.Cmd) {
	if depName, toolName := m.checkRequiredDep(); depName != "" {
		m.missingDepName = depName
		m.missingDepToolName = toolName
		m.pendingConvertCmd = m.doBatchConvert()
		m.isBatchPending = true
		m.state = stateMissingDep
		m.cursor = 0
		return m, nil
	}

	m.browserDir = m.defaultOutput
	m.loadBrowserItems()
	m.state = stateBatchBrowser
	m.cursor = 0
	return m, nil
}

func (m interactiveModel) proceedAfterResizeSelection() (tea.Model, tea.Cmd) {
	if m.resizeIsBatchFlow {
		return m.goToBatchBrowserOrDependencyCheck()
	}
	return m.goToFileBrowser(), nil
}

func (m interactiveModel) buildResizeSpecFromSelection() (*converter.ResizeSpec, error) {
	switch m.resizeMethod {
	case "", "none":
		return nil, nil
	case "preset":
		if strings.TrimSpace(m.resizePresetName) == "" {
			return nil, fmt.Errorf("lütfen bir preset seçin")
		}
		return converter.BuildResizeSpec(m.resizePresetName, 0, 0, "px", m.resizeModeName, 96)
	case "manual":
		width, err := parseResizeInputValue(m.resizeWidthInput)
		if err != nil {
			return nil, fmt.Errorf("geçersiz genişlik")
		}
		height, err := parseResizeInputValue(m.resizeHeightInput)
		if err != nil {
			return nil, fmt.Errorf("geçersiz yükseklik")
		}
		unit := m.resizeUnit
		if unit == "" {
			unit = "px"
		}
		dpi := 96.0
		if unit == "cm" {
			dpi, err = parseResizeInputValue(m.resizeDPIInput)
			if err != nil {
				return nil, fmt.Errorf("geçersiz DPI")
			}
		}
		return converter.BuildResizeSpec("", width, height, unit, m.resizeModeName, dpi)
	default:
		return nil, fmt.Errorf("geçersiz boyutlandırma yöntemi")
	}
}

func parseResizeInputValue(raw string) (float64, error) {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return 0, fmt.Errorf("boş değer")
	}
	normalized = strings.ReplaceAll(normalized, ",", ".")
	v, err := strconv.ParseFloat(normalized, 64)
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("geçersiz sayı")
	}
	return v, nil
}

func (m interactiveModel) isResizeTextInputState() bool {
	switch m.state {
	case stateResizeManualWidth, stateResizeManualHeight, stateResizeManualDPI:
		return true
	default:
		return false
	}
}

func (m *interactiveModel) appendResizeInput(token string) bool {
	field := m.currentResizeInputField()
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
	if ch == ',' || ch == '.' {
		if strings.Contains(*field, ".") {
			return true
		}
		if strings.Contains(*field, ",") {
			return true
		}
		*field += "."
		return true
	}
	return false
}

func (m *interactiveModel) popResizeInput() {
	field := m.currentResizeInputField()
	if field == nil || *field == "" {
		return
	}
	runes := []rune(*field)
	*field = string(runes[:len(runes)-1])
}

func (m *interactiveModel) currentResizeInputField() *string {
	switch m.state {
	case stateResizeManualWidth:
		return &m.resizeWidthInput
	case stateResizeManualHeight:
		return &m.resizeHeightInput
	case stateResizeManualDPI:
		return &m.resizeDPIInput
	default:
		return nil
	}
}

func (m interactiveModel) resizeSummary() string {
	if m.resizeSpec == nil {
		return "Kapalı"
	}
	source := "manuel"
	if m.resizeSpec.Preset != "" {
		source = "preset: " + m.resizeSpec.Preset
	}
	return fmt.Sprintf("%dx%d (%s, mod: %s)", m.resizeSpec.Width, m.resizeSpec.Height, source, m.resizeSpec.Mode)
}

func (m interactiveModel) viewResizeConfig() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" ◆ Boyutlandırma Ayarı "))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		label := menuLine(m.choiceIcons[i], choice)
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
	b.WriteString(infoStyle.Render(fmt.Sprintf("  Seçili: %s", m.resizeSummary())))
	b.WriteString("\n")
	if m.resizeValidationErr != "" {
		b.WriteString(errorStyle.Render("  Hata: " + m.resizeValidationErr))
		b.WriteString("\n")
	}
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç  •  Esc Geri"))
	b.WriteString("\n")
	return b.String()
}

func (m interactiveModel) viewResizePresetSelect() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" ◆ Hazır Boyut (Preset) Seçin "))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		label := menuLine(m.choiceIcons[i], choice)
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

	if m.resizeValidationErr != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("  Hata: " + m.resizeValidationErr))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç  •  Esc Geri"))
	b.WriteString("\n")
	return b.String()
}

func (m interactiveModel) viewResizeNumericInput(title string, value string, hint string) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(fmt.Sprintf(" ◆ %s ", title)))
	b.WriteString("\n\n")

	cursor := " "
	if m.showCursor {
		cursor = "▌"
	}

	input := value
	if input == "" {
		input = ""
	}

	b.WriteString(pathStyle.Render(fmt.Sprintf("  > %s%s", input, cursor)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  " + hint))
	b.WriteString("\n")

	if m.resizeValidationErr != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("  Hata: " + m.resizeValidationErr))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Sayı gir  •  Backspace Sil  •  Enter Devam  •  Esc Geri"))
	b.WriteString("\n")
	return b.String()
}

func (m interactiveModel) viewResizeUnitSelect() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" ◆ Ölçü Birimi Seçin "))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		label := menuLine(m.choiceIcons[i], choice)
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

	if m.resizeValidationErr != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("  Hata: " + m.resizeValidationErr))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Seç  •  Esc Geri"))
	b.WriteString("\n")
	return b.String()
}

func (m interactiveModel) viewResizeModeSelect() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(menuTitleStyle.Render(" ◆ Boyutlandırma Modu Seçin "))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		label := menuLine(m.choiceIcons[i], choice)
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

	if m.resizeMethod == "preset" && m.resizePresetName != "" {
		b.WriteString("\n")
		b.WriteString(infoStyle.Render(fmt.Sprintf("  Preset: %s", strings.ToUpper(m.resizePresetName))))
		b.WriteString("\n")
	}
	if m.resizeMethod == "manual" {
		b.WriteString("\n")
		b.WriteString(infoStyle.Render(fmt.Sprintf("  Manuel: %sx%s %s", m.resizeWidthInput, m.resizeHeightInput, strings.ToUpper(m.resizeUnit))))
		b.WriteString("\n")
	}

	if m.resizeValidationErr != "" {
		b.WriteString(errorStyle.Render("  Hata: " + m.resizeValidationErr))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑↓ Gezin  •  Enter Onayla  •  Esc Geri"))
	b.WriteString("\n")
	return b.String()
}
