package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const projectConfigFileName = ".fileconverter.toml"

// ProjectConfig proje bazlı CLI varsayılanlarını tutar.
type ProjectConfig struct {
	DefaultOutput string
	Workers       int
	Quality       int
	OnConflict    string
	Retry         int
	RetryDelay    time.Duration
	ReportFormat  string
}

// LoadProjectConfig currentDir'den yukarı doğru .fileconverter.toml arar.
// Dosya yoksa (nil, "", nil) döner.
func LoadProjectConfig(currentDir string) (*ProjectConfig, string, error) {
	path, err := findProjectConfigPath(currentDir)
	if err != nil {
		return nil, "", err
	}
	if path == "" {
		return nil, "", nil
	}

	cfg, err := parseProjectConfig(path)
	if err != nil {
		return nil, "", err
	}
	return cfg, path, nil
}

func findProjectConfigPath(startDir string) (string, error) {
	if strings.TrimSpace(startDir) == "" {
		return "", errors.New("gecersiz calisma dizini")
	}

	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, projectConfigFileName)
		info, statErr := os.Stat(candidate)
		if statErr == nil && !info.IsDir() {
			return candidate, nil
		}
		if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			return "", statErr
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", nil
}

func parseProjectConfig(path string) (*ProjectConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := &ProjectConfig{}
	scanner := bufio.NewScanner(f)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(stripInlineComment(scanner.Text()))
		if line == "" {
			continue
		}
		// Bu sürümde sadece top-level key/value desteklenir.
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("%s:%d gecersiz satir", path, lineNo)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			return nil, fmt.Errorf("%s:%d gecersiz key/value", path, lineNo)
		}

		if err := assignProjectConfigValue(cfg, key, value); err != nil {
			return nil, fmt.Errorf("%s:%d %w", path, lineNo, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if cfg.Workers < 0 {
		return nil, fmt.Errorf("workers 0 veya daha buyuk olmali")
	}
	if cfg.Quality < 0 || cfg.Quality > 100 {
		return nil, fmt.Errorf("quality 0-100 araliginda olmali")
	}
	if cfg.Retry < 0 {
		return nil, fmt.Errorf("retry 0 veya daha buyuk olmali")
	}
	if cfg.RetryDelay < 0 {
		return nil, fmt.Errorf("retry_delay negatif olamaz")
	}

	return cfg, nil
}

func assignProjectConfigValue(cfg *ProjectConfig, key, rawValue string) error {
	switch key {
	case "default_output":
		v, err := parseTomlString(rawValue)
		if err != nil {
			return err
		}
		cfg.DefaultOutput = v
	case "workers":
		v, err := parseTomlInt(rawValue)
		if err != nil {
			return err
		}
		cfg.Workers = v
	case "quality":
		v, err := parseTomlInt(rawValue)
		if err != nil {
			return err
		}
		cfg.Quality = v
	case "on_conflict":
		v, err := parseTomlString(rawValue)
		if err != nil {
			return err
		}
		cfg.OnConflict = strings.ToLower(strings.TrimSpace(v))
	case "retry":
		v, err := parseTomlInt(rawValue)
		if err != nil {
			return err
		}
		cfg.Retry = v
	case "retry_delay":
		v, err := parseTomlDuration(rawValue)
		if err != nil {
			return err
		}
		cfg.RetryDelay = v
	case "report_format":
		v, err := parseTomlString(rawValue)
		if err != nil {
			return err
		}
		cfg.ReportFormat = strings.ToLower(strings.TrimSpace(v))
	default:
		// Bilinmeyen anahtarları görmezden gel.
	}
	return nil
}

func parseTomlString(v string) (string, error) {
	v = strings.TrimSpace(v)
	if len(v) < 2 {
		return "", fmt.Errorf("gecersiz string deger")
	}
	if (strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"")) ||
		(strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'")) {
		return v[1 : len(v)-1], nil
	}
	// Kısa yazım: key = value (tırnaksız)
	return v, nil
}

func parseTomlInt(v string) (int, error) {
	v = strings.TrimSpace(v)
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("gecersiz sayi degeri")
	}
	return parsed, nil
}

func parseTomlDuration(v string) (time.Duration, error) {
	str, err := parseTomlString(v)
	if err != nil {
		return 0, err
	}
	d, err := time.ParseDuration(str)
	if err != nil {
		return 0, fmt.Errorf("gecersiz sure degeri")
	}
	return d, nil
}

func stripInlineComment(line string) string {
	inSingle := false
	inDouble := false

	for i, r := range line {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble {
				return line[:i]
			}
		}
	}
	return line
}
