package main

import (
	"os"
	"runtime/debug"
	"strings"

	"github.com/mlihgenel/fileconverter-cli/cmd"
)

var (
	// ldflags ile override edilebilir:
	//   -X main.version=1.5.0 -X main.buildDate=2026-02-21T12:00:00Z
	version   = "dev"
	buildDate = ""
)

func main() {
	cmd.SetVersionInfo(resolveVersion(), resolveBuildDate())
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func resolveVersion() string {
	v := normalizeVersion(version)
	if v != "" && v != "dev" {
		return v
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if moduleVersion := normalizeVersion(info.Main.Version); moduleVersion != "" && moduleVersion != "(devel)" {
		return moduleVersion
	}

	revision := strings.TrimSpace(buildSetting(info, "vcs.revision"))
	if revision == "" {
		return "dev"
	}
	if len(revision) > 7 {
		revision = revision[:7]
	}
	if strings.EqualFold(strings.TrimSpace(buildSetting(info, "vcs.modified")), "true") {
		return "dev-" + revision + "-dirty"
	}
	return "dev-" + revision
}

func resolveBuildDate() string {
	if v := strings.TrimSpace(buildDate); v != "" {
		return v
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return strings.TrimSpace(buildSetting(info, "vcs.time"))
}

func buildSetting(info *debug.BuildInfo, key string) string {
	for _, setting := range info.Settings {
		if setting.Key == key {
			return setting.Value
		}
	}
	return ""
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "v") {
		return strings.TrimPrefix(v, "v")
	}
	return v
}
