package cmd

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	envOutput     = "FILECONVERTER_OUTPUT"
	envWorkers    = "FILECONVERTER_WORKERS"
	envQuality    = "FILECONVERTER_QUALITY"
	envConflict   = "FILECONVERTER_ON_CONFLICT"
	envRetry      = "FILECONVERTER_RETRY"
	envRetryDelay = "FILECONVERTER_RETRY_DELAY"
	envReport     = "FILECONVERTER_REPORT"
)

func applyRootDefaults(cmd *cobra.Command) error {
	if !cmd.Flags().Changed("output") {
		if v := strings.TrimSpace(os.Getenv(envOutput)); v != "" {
			outputDir = v
		} else if activeProjectConfig != nil && strings.TrimSpace(activeProjectConfig.DefaultOutput) != "" {
			outputDir = strings.TrimSpace(activeProjectConfig.DefaultOutput)
		}
	}

	if !cmd.Flags().Changed("workers") {
		if v, ok := readEnvInt(envWorkers); ok && v > 0 {
			workers = v
		} else if activeProjectConfig != nil && activeProjectConfig.Workers > 0 {
			workers = activeProjectConfig.Workers
		}
	}

	return nil
}

func applyQualityDefault(cmd *cobra.Command, flagName string, value *int) {
	if cmd.Flags().Changed(flagName) {
		return
	}
	if v, ok := readEnvInt(envQuality); ok && v >= 0 {
		*value = v
		return
	}
	if activeProjectConfig != nil && activeProjectConfig.Quality > 0 {
		*value = activeProjectConfig.Quality
	}
}

func applyOnConflictDefault(cmd *cobra.Command, flagName string, value *string) {
	if cmd.Flags().Changed(flagName) {
		return
	}
	if v := strings.TrimSpace(os.Getenv(envConflict)); v != "" {
		*value = strings.ToLower(v)
		return
	}
	if activeProjectConfig != nil && strings.TrimSpace(activeProjectConfig.OnConflict) != "" {
		*value = strings.ToLower(strings.TrimSpace(activeProjectConfig.OnConflict))
	}
}

func applyRetryDefaults(cmd *cobra.Command, retryFlag string, retryValue *int, delayFlag string, delayValue *time.Duration) {
	if !cmd.Flags().Changed(retryFlag) {
		if v, ok := readEnvInt(envRetry); ok && v >= 0 {
			*retryValue = v
		} else if activeProjectConfig != nil && activeProjectConfig.Retry > 0 {
			*retryValue = activeProjectConfig.Retry
		}
	}

	if !cmd.Flags().Changed(delayFlag) {
		if v, ok := readEnvDuration(envRetryDelay); ok {
			*delayValue = v
		} else if activeProjectConfig != nil && activeProjectConfig.RetryDelay > 0 {
			*delayValue = activeProjectConfig.RetryDelay
		}
	}
}

func applyReportDefault(cmd *cobra.Command, flagName string, value *string) {
	if cmd.Flags().Changed(flagName) {
		return
	}
	if v := strings.TrimSpace(os.Getenv(envReport)); v != "" {
		*value = strings.ToLower(v)
		return
	}
	if activeProjectConfig != nil && strings.TrimSpace(activeProjectConfig.ReportFormat) != "" {
		*value = strings.ToLower(strings.TrimSpace(activeProjectConfig.ReportFormat))
	}
}

func readEnvInt(name string) (int, bool) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return 0, false
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return v, true
}

func readEnvDuration(name string) (time.Duration, bool) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return 0, false
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, false
	}
	return v, true
}
