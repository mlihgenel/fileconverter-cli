package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mlihgenel/fileconverter-cli/internal/converter"
	"github.com/mlihgenel/fileconverter-cli/internal/profile"
)

func resolveProfile(name string) (profile.Definition, bool, error) {
	if name == "" {
		return profile.Definition{}, false, nil
	}
	p, err := profile.Resolve(name)
	if err != nil {
		return profile.Definition{}, false, err
	}
	return p, true, nil
}

func applyProfileToConvert(cmd *cobra.Command, p profile.Definition) {
	if p.Quality != nil && !cmd.Flags().Changed("quality") {
		quality = *p.Quality
	}
	if p.OnConflict != "" && !cmd.Flags().Changed("on-conflict") {
		convertOnConflict = p.OnConflict
	}
	if p.ResizePreset != "" && !cmd.Flags().Changed("preset") {
		convertPreset = p.ResizePreset
	}
	if p.ResizeMode != "" && !cmd.Flags().Changed("resize-mode") {
		convertResizeMode = p.ResizeMode
	}
	if p.Width != nil && !cmd.Flags().Changed("width") {
		convertWidth = *p.Width
	}
	if p.Height != nil && !cmd.Flags().Changed("height") {
		convertHeight = *p.Height
	}
	if p.Unit != "" && !cmd.Flags().Changed("unit") {
		convertUnit = p.Unit
	}
	if p.DPI != nil && !cmd.Flags().Changed("dpi") {
		convertResizeDPI = *p.DPI
	}
}

func applyProfileToBatch(cmd *cobra.Command, p profile.Definition) {
	if p.Quality != nil && !cmd.Flags().Changed("quality") {
		batchQuality = *p.Quality
	}
	if p.OnConflict != "" && !cmd.Flags().Changed("on-conflict") {
		batchOnConflict = p.OnConflict
	}
	if p.Retry != nil && !cmd.Flags().Changed("retry") {
		batchRetry = *p.Retry
	}
	if p.RetryDelay != nil && !cmd.Flags().Changed("retry-delay") {
		batchRetryDelay = *p.RetryDelay
	}
	if p.Report != "" && !cmd.Flags().Changed("report") {
		batchReport = p.Report
	}
	if p.ResizePreset != "" && !cmd.Flags().Changed("preset") {
		batchPreset = p.ResizePreset
	}
	if p.ResizeMode != "" && !cmd.Flags().Changed("resize-mode") {
		batchResizeMode = p.ResizeMode
	}
	if p.Width != nil && !cmd.Flags().Changed("width") {
		batchWidth = *p.Width
	}
	if p.Height != nil && !cmd.Flags().Changed("height") {
		batchHeight = *p.Height
	}
	if p.Unit != "" && !cmd.Flags().Changed("unit") {
		batchUnit = p.Unit
	}
	if p.DPI != nil && !cmd.Flags().Changed("dpi") {
		batchResizeDPI = *p.DPI
	}
}

func applyProfileToWatch(cmd *cobra.Command, p profile.Definition) {
	if p.Quality != nil && !cmd.Flags().Changed("quality") {
		watchQuality = *p.Quality
	}
	if p.OnConflict != "" && !cmd.Flags().Changed("on-conflict") {
		watchOnConflict = p.OnConflict
	}
	if p.Retry != nil && !cmd.Flags().Changed("retry") {
		watchRetry = *p.Retry
	}
	if p.RetryDelay != nil && !cmd.Flags().Changed("retry-delay") {
		watchRetryDelay = *p.RetryDelay
	}
}

func applyProfileToPipeline(cmd *cobra.Command, p profile.Definition) {
	if p.Quality != nil && !cmd.Flags().Changed("quality") {
		pipelineQuality = *p.Quality
	}
	if p.OnConflict != "" && !cmd.Flags().Changed("on-conflict") {
		pipelineOnConflict = p.OnConflict
	}
	if p.Report != "" && !cmd.Flags().Changed("report") {
		pipelineReport = p.Report
	}
}

func applyProfileMetadata(cmd *cobra.Command, p profile.Definition, preserveFlag string, preserveValue *bool, stripFlag string, stripValue *bool) {
	if p.MetadataMode == "" {
		return
	}
	if cmd.Flags().Changed(preserveFlag) || cmd.Flags().Changed(stripFlag) {
		return
	}
	switch converter.NormalizeMetadataMode(p.MetadataMode) {
	case converter.MetadataPreserve:
		*preserveValue = true
		*stripValue = false
	case converter.MetadataStrip:
		*stripValue = true
		*preserveValue = false
	}
}

func metadataModeFromFlags(preserve bool, strip bool) (string, error) {
	if preserve && strip {
		return "", fmt.Errorf("--preserve-metadata ve --strip-metadata birlikte kullanılamaz")
	}
	if preserve {
		return converter.MetadataPreserve, nil
	}
	if strip {
		return converter.MetadataStrip, nil
	}
	return converter.MetadataAuto, nil
}
