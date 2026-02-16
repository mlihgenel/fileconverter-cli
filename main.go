package main

import (
	"os"

	"github.com/melihgenel/fileconverter-cli/cmd"
)

// Version bilgisi build sırasında enjekte edilir
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, date)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
