package cmd

import (
	"strings"
)

func (m interactiveModel) isSprint2TextInputState() bool {
	switch m.state {
	case stateSnapshotTime:
		return true
	default:
		return false
	}
}

func (m *interactiveModel) currentSprint2InputField() *string {
	switch m.state {
	case stateSnapshotTime:
		return &m.snapshotTimeInput
	default:
		return nil
	}
}

func (m *interactiveModel) appendSprint2Input(token string) bool {
	field := m.currentSprint2InputField()
	if field == nil {
		return false
	}

	r := []rune(token)
	if len(r) != 1 {
		return false
	}

	ch := r[0]
	// Allows digits, minus sign, colon, percentage and dot/comma
	if (ch >= '0' && ch <= '9') || ch == '-' || ch == ':' || ch == '%' {
		*field += string(ch)
		return true
	}
	if ch == '.' || ch == ',' {
		*field += "."
		return true
	}
	return false
}

func (m *interactiveModel) popSprint2Input() {
	field := m.currentSprint2InputField()
	if field == nil || *field == "" {
		return
	}
	runes := []rune(*field)
	*field = string(runes[:len(runes)-1])
}

// Merge file selection logic
func (m *interactiveModel) toggleMergeFileSelection() {
	if m.cursor >= len(m.browserItems) {
		return
	}
	item := m.browserItems[m.cursor]
	if item.isDir {
		return // Cannot merge directores
	}

	// Check if already selected
	idx := -1
	for i, f := range m.mergeFiles {
		if f == item.path {
			idx = i
			break
		}
	}

	if idx >= 0 {
		// Remove
		m.mergeFiles = append(m.mergeFiles[:idx], m.mergeFiles[idx+1:]...)
	} else {
		// Add
		m.mergeFiles = append(m.mergeFiles, item.path)
	}
}

// Missing goToMergeBrowser from cmd/interactive_merge.go
func (m interactiveModel) goToMergeBrowser() interactiveModel {
	m.flowIsBatch = false
	m.flowResizeOnly = false
	m.flowIsWatch = false
	m.flowVideoTrim = false
	m.flowExtractAudio = false
	m.flowSnapshot = false
	m.flowMerge = true
	m.flowAudioNormalize = false
	m.resetResizeState()
	m.sourceFormat = ""
	m.targetFormat = ""
	m.mergeFiles = nil
	m.selectedCategory = videoCategoryIndex()

	m.mergeQualityInput = "0"
	m.mergeReencodeMode = false

	m.state = stateMergeBrowser
	m.cursor = 0
	if strings.TrimSpace(m.browserDir) == "" {
		m.browserDir = m.defaultOutput
	}
	m.loadBrowserItems()
	return m
}
