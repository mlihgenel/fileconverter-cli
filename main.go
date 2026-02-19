package main

import (
	"os"

	"github.com/mlihgenel/fileconverter-cli/cmd"
)

var (
	version = "1.2.0"
)

func main() {
	cmd.SetVersionInfo(version, "")
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
