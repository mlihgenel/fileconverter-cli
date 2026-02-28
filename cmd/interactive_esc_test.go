package cmd

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestGoBackFromCategorySelectReturnsParentSection(t *testing.T) {
	m := newInteractiveModel(nil, false)
	m.mainSection = "conversion"
	m.state = stateSelectCategory

	next := m.goBack()
	if next.state != stateMainSectionMenu {
		t.Fatalf("expected stateMainSectionMenu, got %v", next.state)
	}
	if next.mainSection != "conversion" {
		t.Fatalf("expected conversion section, got %s", next.mainSection)
	}
	if len(next.choices) == 0 || next.choices[0] != "Tek Dosya Dönüştür" {
		t.Fatalf("unexpected section choices: %+v", next.choices)
	}
}

func TestGoBackFromSpecialFileBrowserReturnsParentSection(t *testing.T) {
	m := newInteractiveModel(nil, false)
	m.mainSection = "video"
	m.state = stateFileBrowser
	m.flowExtractAudio = true

	next := m.goBack()
	if next.state != stateMainSectionMenu {
		t.Fatalf("expected stateMainSectionMenu, got %v", next.state)
	}
	if next.mainSection != "video" {
		t.Fatalf("expected video section, got %s", next.mainSection)
	}
	if len(next.choices) == 0 || next.choices[0] != "Video Düzenle (Klip/Sil)" {
		t.Fatalf("unexpected section choices: %+v", next.choices)
	}
}

func TestGoBackFromSystemScreenReturnsParentSection(t *testing.T) {
	m := newInteractiveModel(nil, false)
	m.mainSection = "system"
	m.state = stateFormats

	next := m.goBack()
	if next.state != stateMainSectionMenu {
		t.Fatalf("expected stateMainSectionMenu, got %v", next.state)
	}
	if next.mainSection != "system" {
		t.Fatalf("expected system section, got %s", next.mainSection)
	}
	if len(next.choices) == 0 || next.choices[0] != "Dosya Bilgisi" {
		t.Fatalf("unexpected section choices: %+v", next.choices)
	}
}

func TestGoBackFromMissingDependencyReturnsFileBrowser(t *testing.T) {
	dir := t.TempDir()
	m := newInteractiveModel(nil, false)
	m.mainSection = "video"
	m.state = stateMissingDep
	m.flowSnapshot = true
	m.browserDir = dir
	m.selectedCategory = videoCategoryIndex()

	next := m.goBack()
	if next.state != stateFileBrowser {
		t.Fatalf("expected stateFileBrowser, got %v", next.state)
	}
}

func TestEscKeyFromSnapshotTimeReturnsFileBrowser(t *testing.T) {
	dir := t.TempDir()
	m := newInteractiveModel(nil, false)
	m.mainSection = "video"
	m.state = stateSnapshotTime
	m.flowSnapshot = true
	m.browserDir = dir
	m.selectedCategory = videoCategoryIndex()

	nextModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatalf("expected no command on esc")
	}
	next, ok := nextModel.(interactiveModel)
	if !ok {
		t.Fatalf("unexpected model type")
	}
	if next.state != stateFileBrowser {
		t.Fatalf("expected stateFileBrowser, got %v", next.state)
	}
}
