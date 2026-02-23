package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	OutputFormatText = "text"
	OutputFormatJSON = "json"
)

func NormalizeOutputFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", OutputFormatText:
		return OutputFormatText
	case OutputFormatJSON:
		return OutputFormatJSON
	default:
		return ""
	}
}

func isJSONOutput() bool {
	return NormalizeOutputFormat(outputFormat) == OutputFormatJSON
}

func outputFormatError(format string) error {
	return fmt.Errorf("gecersiz output-format: %s (text|json)", format)
}

func printJSON(payload any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
