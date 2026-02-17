package main

import (
	"os"

	"github.com/mlihgenel/fileconverter-cli/cmd"
)

var (
	version = "1.0.0"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, date)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
