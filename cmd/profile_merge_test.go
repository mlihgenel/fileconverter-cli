package cmd

import (
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlihgenel/fileconverter-cli/internal/profile"
)

func TestMetadataModeFromFlags(t *testing.T) {
	mode, err := metadataModeFromFlags(false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "auto" {
		t.Fatalf("expected auto, got %s", mode)
	}

	mode, err = metadataModeFromFlags(true, false)
	if err != nil || mode != "preserve" {
		t.Fatalf("expected preserve, got mode=%s err=%v", mode, err)
	}

	mode, err = metadataModeFromFlags(false, true)
	if err != nil || mode != "strip" {
		t.Fatalf("expected strip, got mode=%s err=%v", mode, err)
	}

	if _, err := metadataModeFromFlags(true, true); err == nil {
		t.Fatalf("expected error when both preserve and strip set")
	}
}

func TestApplyProfileMetadata(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("preserve-metadata", false, "")
	cmd.Flags().Bool("strip-metadata", false, "")

	preserve := false
	strip := false
	p := profile.Definition{MetadataMode: "strip"}
	applyProfileMetadata(cmd, p, "preserve-metadata", &preserve, "strip-metadata", &strip)
	if !strip || preserve {
		t.Fatalf("expected strip=true preserve=false, got strip=%v preserve=%v", strip, preserve)
	}
}

func TestApplyProfileToBatch(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Int("quality", 0, "")
	cmd.Flags().String("on-conflict", "", "")
	cmd.Flags().Int("retry", 0, "")
	cmd.Flags().Duration("retry-delay", 0, "")
	cmd.Flags().String("report", "", "")
	cmd.Flags().String("preset", "", "")
	cmd.Flags().String("resize-mode", "", "")
	cmd.Flags().Float64("width", 0, "")
	cmd.Flags().Float64("height", 0, "")
	cmd.Flags().String("unit", "", "")
	cmd.Flags().Float64("dpi", 0, "")

	// Reset globals used by applyProfileToBatch
	batchQuality = 0
	batchOnConflict = ""
	batchRetry = 0
	batchRetryDelay = 0
	batchReport = ""
	batchPreset = ""
	batchResizeMode = ""
	batchWidth = 0
	batchHeight = 0
	batchUnit = ""
	batchResizeDPI = 0

	p := profile.Definition{
		Quality:      profile.IntPtr(77),
		OnConflict:   "versioned",
		Retry:        profile.IntPtr(2),
		RetryDelay:   profile.DurationPtr(2 * time.Second),
		Report:       "json",
		ResizePreset: "story",
		ResizeMode:   "pad",
		Width:        profile.FloatPtr(1080),
		Height:       profile.FloatPtr(1920),
		Unit:         "px",
		DPI:          profile.FloatPtr(96),
	}

	applyProfileToBatch(cmd, p)
	if batchQuality != 77 || batchOnConflict != "versioned" || batchRetry != 2 || batchReport != "json" {
		t.Fatalf("batch profile values did not apply")
	}
	if batchRetryDelay != 2*time.Second {
		t.Fatalf("expected retry delay 2s, got %s", batchRetryDelay)
	}
}
