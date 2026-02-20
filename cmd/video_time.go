package cmd

import (
	"fmt"
	"strconv"
	"strings"
)

func normalizeVideoTrimTime(raw string, allowZero bool) (string, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(raw), ",", ".")
	if normalized == "" {
		return "", fmt.Errorf("boş değer")
	}

	seconds, err := parseVideoTrimToSeconds(normalized)
	if err != nil {
		return "", err
	}
	if !allowZero && seconds <= 0 {
		return "", fmt.Errorf("süre sıfırdan büyük olmalı")
	}
	if allowZero && seconds < 0 {
		return "", fmt.Errorf("değer negatif olamaz")
	}

	if strings.Contains(normalized, ":") {
		parts := strings.Split(normalized, ":")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return strings.Join(parts, ":"), nil
	}
	return strconv.FormatFloat(seconds, 'f', -1, 64), nil
}

func parseVideoTrimToSeconds(value string) (float64, error) {
	normalized := strings.TrimSpace(value)
	if strings.Contains(normalized, ":") {
		parts := strings.Split(normalized, ":")
		if len(parts) < 2 || len(parts) > 3 {
			return 0, fmt.Errorf("zaman formatı hatalı")
		}

		parsed := make([]float64, len(parts))
		for i, part := range parts {
			p := strings.TrimSpace(part)
			if p == "" {
				return 0, fmt.Errorf("zaman formatı hatalı")
			}
			v, err := strconv.ParseFloat(p, 64)
			if err != nil || v < 0 {
				return 0, fmt.Errorf("zaman formatı hatalı")
			}
			parsed[i] = v
		}

		if len(parsed) == 2 {
			if parsed[1] >= 60 {
				return 0, fmt.Errorf("saniye 60'tan küçük olmalı")
			}
			return parsed[0]*60 + parsed[1], nil
		}

		if parsed[1] >= 60 || parsed[2] >= 60 {
			return 0, fmt.Errorf("dakika/saniye 60'tan küçük olmalı")
		}
		return parsed[0]*3600 + parsed[1]*60 + parsed[2], nil
	}

	v, err := strconv.ParseFloat(normalized, 64)
	if err != nil || v < 0 {
		return 0, fmt.Errorf("geçersiz sayı")
	}
	return v, nil
}
