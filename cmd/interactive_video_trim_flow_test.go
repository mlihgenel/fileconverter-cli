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
	m.trimRangeType = trimRangeDuration
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

func TestVideoTrimStartToRangeTypeTransition(t *testing.T) {
	m := newInteractiveModel(nil, false)
	m.state = stateVideoTrimStart
	m.trimStartInput = "23"
	m.trimRangeType = trimRangeDuration

	nextModel, cmd := m.handleEnter()
	if cmd != nil {
		t.Fatalf("expected no async command for start step")
	}

	next, ok := nextModel.(interactiveModel)
	if !ok {
		t.Fatalf("unexpected model type")
	}
	if next.state != stateVideoTrimRangeType {
		t.Fatalf("expected stateVideoTrimRangeType, got %v", next.state)
	}
	if len(next.choices) != 2 {
		t.Fatalf("expected 2 range type options")
	}
}

func TestVideoTrimRangeTypeEndSelection(t *testing.T) {
	m := newInteractiveModel(nil, false)
	m.state = stateVideoTrimRangeType
	m.cursor = 1
	m.trimStartInput = "23"

	nextModel, cmd := m.handleEnter()
	if cmd != nil {
		t.Fatalf("expected no async command for range type step")
	}

	next, ok := nextModel.(interactiveModel)
	if !ok {
		t.Fatalf("unexpected model type")
	}
	if next.trimRangeType != trimRangeEnd {
		t.Fatalf("expected end range type, got %s", next.trimRangeType)
	}
	if next.state != stateVideoTrimDuration {
		t.Fatalf("expected stateVideoTrimDuration, got %v", next.state)
	}
	if strings.TrimSpace(next.trimEndInput) == "" {
		t.Fatalf("expected suggested end input")
	}
}

func TestVideoTrimCodecShowsPreview(t *testing.T) {
	m := newInteractiveModel(nil, false)
	m.state = stateVideoTrimCodec
	m.selectedFile = "/tmp/sample.mp4"
	m.trimMode = trimModeClip
	m.trimStartInput = "0"
	m.trimDurationInput = "2"
	m.cursor = 0

	nextModel, cmd := m.handleEnter()
	if cmd != nil {
		t.Fatalf("expected no async command on preview step")
	}

	next, ok := nextModel.(interactiveModel)
	if !ok {
		t.Fatalf("unexpected model type")
	}
	if next.state != stateVideoTrimPreview {
		t.Fatalf("expected stateVideoTrimPreview, got %v", next.state)
	}
	if next.targetFormat != "mp4" {
		t.Fatalf("expected detected target format mp4, got %s", next.targetFormat)
	}
	if next.trimPreviewPlan == nil {
		t.Fatalf("expected trim preview plan to be prepared")
	}
}

func TestVideoTrimPreviewStartsConverting(t *testing.T) {
	m := newInteractiveModel(nil, false)
	m.state = stateVideoTrimPreview
	m.selectedFile = "/tmp/sample.mp4"
	m.trimMode = trimModeClip
	m.trimStartInput = "0"
	m.trimDurationInput = "2"
	m.trimCodec = "copy"
	m.cursor = 0

	nextModel, cmd := m.handleEnter()
	if cmd == nil {
		t.Fatalf("expected conversion command from preview")
	}

	next, ok := nextModel.(interactiveModel)
	if !ok {
		t.Fatalf("unexpected model type")
	}
	if next.state != stateConverting {
		t.Fatalf("expected stateConverting, got %v", next.state)
	}
}

func TestVideoTrimPreviewBackToCodec(t *testing.T) {
	m := newInteractiveModel(nil, false)
	m.state = stateVideoTrimPreview
	m.cursor = 1

	nextModel, cmd := m.handleEnter()
	if cmd != nil {
		t.Fatalf("expected no async command when returning to codec")
	}

	next, ok := nextModel.(interactiveModel)
	if !ok {
		t.Fatalf("unexpected model type")
	}
	if next.state != stateVideoTrimCodec {
		t.Fatalf("expected stateVideoTrimCodec, got %v", next.state)
	}
}
