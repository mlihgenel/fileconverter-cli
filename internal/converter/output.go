package converter

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ConflictOverwrite = "overwrite"
	ConflictSkip      = "skip"
	ConflictVersioned = "versioned"
)

// NormalizeConflictPolicy geçersiz/boş değerlerde varsayılan policy döner.
func NormalizeConflictPolicy(policy string) string {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case ConflictOverwrite:
		return ConflictOverwrite
	case ConflictSkip:
		return ConflictSkip
	case ConflictVersioned, "":
		return ConflictVersioned
	default:
		return ""
	}
}

// ResolveOutputPathConflict hedef dosya adı çakışmasını verilen policy'ye göre çözer.
// skip=true dönerse ilgili iş atlanmalıdır.
func ResolveOutputPathConflict(path, policy string) (resolvedPath string, skip bool, err error) {
	normalized := NormalizeConflictPolicy(policy)
	if normalized == "" {
		return "", false, fmt.Errorf("gecersiz on-conflict politikasi: %s", policy)
	}

	_, statErr := os.Stat(path)
	if statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return path, false, nil
		}
		return "", false, statErr
	}

	switch normalized {
	case ConflictOverwrite:
		return path, false, nil
	case ConflictSkip:
		return path, true, nil
	case ConflictVersioned:
		ext := filepath.Ext(path)
		base := strings.TrimSuffix(path, ext)
		for i := 1; i < 100000; i++ {
			candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
			if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
				return candidate, false, nil
			} else if err != nil {
				return "", false, err
			}
		}
		return "", false, fmt.Errorf("uygun versioned dosya adi bulunamadi")
	default:
		return "", false, fmt.Errorf("gecersiz on-conflict politikasi: %s", policy)
	}
}
