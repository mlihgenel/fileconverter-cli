package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectConfigFindsParent(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	cfgPath := filepath.Join(root, projectConfigFileName)
	content := `
default_output = "./out"
workers = 4
quality = 85
on_conflict = "versioned"
retry = 2
retry_delay = "1s"
report_format = "json"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	cfg, foundPath, err := LoadProjectConfig(nested)
	if err != nil {
		t.Fatalf("LoadProjectConfig failed: %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected config, got nil")
	}
	if foundPath != cfgPath {
		t.Fatalf("unexpected config path: %s", foundPath)
	}
	if cfg.Workers != 4 {
		t.Fatalf("unexpected workers: %d", cfg.Workers)
	}
	if cfg.Quality != 85 {
		t.Fatalf("unexpected quality: %d", cfg.Quality)
	}
	if cfg.Retry != 2 {
		t.Fatalf("unexpected retry: %d", cfg.Retry)
	}
	if cfg.ReportFormat != "json" {
		t.Fatalf("unexpected report format: %s", cfg.ReportFormat)
	}
}

func TestLoadProjectConfigMissingFile(t *testing.T) {
	cfg, path, err := LoadProjectConfig(t.TempDir())
	if err != nil {
		t.Fatalf("LoadProjectConfig failed: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config for missing file")
	}
	if path != "" {
		t.Fatalf("expected empty path, got: %s", path)
	}
}

func TestLoadProjectConfigInvalidValue(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, projectConfigFileName)
	content := `quality = 200`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	_, _, err := LoadProjectConfig(root)
	if err == nil {
		t.Fatalf("expected error for invalid quality")
	}
}
