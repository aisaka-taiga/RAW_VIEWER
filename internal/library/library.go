package library

import (
	"path/filepath"
	"strings"
)

func IsSupported(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".arw", ".cr3", ".nef", ".dng", ".rw2", ".heic", ".heif", ".jpg", ".jpeg", ".png", ".tif", ".tiff":
		return true
	default:
		return false
	}
}
