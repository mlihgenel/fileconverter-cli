package cmd

import "testing"

func TestNormalizeOutputFormat(t *testing.T) {
	if got := NormalizeOutputFormat(""); got != OutputFormatText {
		t.Fatalf("expected text for empty, got %s", got)
	}
	if got := NormalizeOutputFormat("TEXT"); got != OutputFormatText {
		t.Fatalf("expected text for TEXT, got %s", got)
	}
	if got := NormalizeOutputFormat("json"); got != OutputFormatJSON {
		t.Fatalf("expected json, got %s", got)
	}
	if got := NormalizeOutputFormat("yaml"); got != "" {
		t.Fatalf("expected empty for invalid format, got %s", got)
	}
}
