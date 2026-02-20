package cmd

import (
	"strings"
	"testing"
)

func TestVideoTrimModeSelectionTransition(t *testing.T) {
	m := newInteractiveModel(nil, false)
	m.state = stateVideoTrimMode
	m.cursor = 1
	m.choices = []string{"Klip Çıkar", "Aralığı Sil + Birleştir"}

	nextModel, cmd := m.handleEnter()
	if cmd != nil {
		t.Fatalf("expected no async command for mode selection")
	}

	next, ok := nextModel.(interactiveModel)
	if !ok {
		t.Fatalf("unexpected model type")
	}
	if next.trimMode != trimModeRemove {
		t.Fatalf("expected remove mode, got %s", next.trimMode)
	}
	if next.state != stateVideoTrimStart {
		t.Fatalf("expected stateVideoTrimStart, got %v", next.state)
	}
}

func TestVideoTrimDurationToCodecDescriptions(t *testing.T) {
	m := newInteractiveModel(nil, false)
	m.state = stateVideoTrimDuration
	m.trimMode = trimModeRemove
	m.trimDurationInput = "2"

	nextModel, cmd := m.handleEnter()
	if cmd != nil {
		t.Fatalf("expected no async command for duration step")
	}

	next, ok := nextModel.(interactiveModel)
	if !ok {
		t.Fatalf("unexpected model type")
	}
	if next.state != stateVideoTrimCodec {
		t.Fatalf("expected stateVideoTrimCodec, got %v", next.state)
	}
	if len(next.choiceDescs) == 0 || !strings.Contains(next.choiceDescs[0], "Aralık silme sonrası") {
		t.Fatalf("expected remove-specific codec description")
	}
}

func TestVideoTrimCodecStartsConverting(t *testing.T) {
	m := newInteractiveModel(nil, false)
	m.state = stateVideoTrimCodec
	m.selectedFile = "/tmp/sample.mp4"
	m.trimMode = trimModeClip
	m.trimStartInput = "0"
	m.trimDurationInput = "2"
	m.cursor = 0

	nextModel, cmd := m.handleEnter()
	if cmd == nil {
		t.Fatalf("expected conversion command")
	}

	next, ok := nextModel.(interactiveModel)
	if !ok {
		t.Fatalf("unexpected model type")
	}
	if next.state != stateConverting {
		t.Fatalf("expected stateConverting, got %v", next.state)
	}
	if next.targetFormat != "mp4" {
		t.Fatalf("expected detected target format mp4, got %s", next.targetFormat)
	}
}
