package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AppConfig uygulama yapılandırmasını tutar
type AppConfig struct {
	FirstRunCompleted bool   `json:"first_run_completed"`
	DefaultOutputDir  string `json:"default_output_dir,omitempty"`
}

// configDir yapılandırma dizinini döner (~/.fileconverter)
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".fileconverter"), nil
}

// configPath yapılandırma dosya yolunu döner
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// LoadConfig yapılandırmayı dosyadan okur
func LoadConfig() (*AppConfig, error) {
	path, err := configPath()
	if err != nil {
		return &AppConfig{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// Dosya yoksa varsayılan config döndür
		return &AppConfig{}, nil
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &AppConfig{}, nil
	}

	return &cfg, nil
}

// SaveConfig yapılandırmayı dosyaya kaydeder
func SaveConfig(cfg *AppConfig) error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(dir, "config.json")
	return os.WriteFile(path, data, 0644)
}

// IsFirstRun uygulamanın ilk kez çalıştırılıp çalıştırılmadığını kontrol eder
func IsFirstRun() bool {
	cfg, _ := LoadConfig()
	return !cfg.FirstRunCompleted
}

// MarkFirstRunDone ilk çalıştırma tamamlandı olarak işaretler
func MarkFirstRunDone() error {
	cfg, _ := LoadConfig()
	cfg.FirstRunCompleted = true
	return SaveConfig(cfg)
}

// GetDefaultOutputDir varsayılan çıktı dizinini döner
func GetDefaultOutputDir() string {
	cfg, _ := LoadConfig()
	return cfg.DefaultOutputDir
}

// SetDefaultOutputDir varsayılan çıktı dizinini kaydeder
func SetDefaultOutputDir(dir string) error {
	cfg, _ := LoadConfig()
	cfg.DefaultOutputDir = dir
	return SaveConfig(cfg)
}
