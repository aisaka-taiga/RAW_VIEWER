package thumbnail

import (
	"context"
	"fmt"
	"os/exec"
)

type ExifToolPreviewer struct {
	ExePath string
}

func DefaultExifToolPreviewer() ExifToolPreviewer {
	return ExifToolPreviewer{ExePath: `C:\workspace\exiftool-13.55_64\exiftool(-k).exe`}
}

func (p ExifToolPreviewer) GetPreviewBytes(ctx context.Context, path string, size int) ([]byte, error) {
	tags := previewTagsBySize(size)
	var lastErr error
	for _, tag := range tags {
		cmd := exec.CommandContext(ctx, p.ExePath, "-q", "-q", "-b", "-"+tag, path)
		out, err := cmd.Output()
		if err == nil && len(out) > 0 {
			return out, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no preview data found")
	}
	return nil, fmt.Errorf("preview extraction failed for %s: %w", path, lastErr)
}

func previewTagsBySize(size int) []string {
	switch {
	case size <= 384:
		return []string{"ThumbnailImage", "PreviewImage", "JpgFromRaw"}
	case size <= 1024:
		return []string{"PreviewImage", "ThumbnailImage", "JpgFromRaw"}
	default:
		return []string{"JpgFromRaw", "PreviewImage", "ThumbnailImage"}
	}
}
