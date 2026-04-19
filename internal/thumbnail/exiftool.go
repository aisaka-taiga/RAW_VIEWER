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

func (p ExifToolPreviewer) GetPreviewBytes(ctx context.Context, path string) ([]byte, error) {
	// Request multiple potential preview tags in one go to save process overhead
	// Use CommandContext to kill the process if the user navigates away
	cmd := exec.CommandContext(ctx, p.ExePath, "-q", "-q", "-b", "-JpgFromRaw", "-PreviewImage", "-ThumbnailImage", path)
	out, err := cmd.Output()
	if err == nil && len(out) > 0 {
		return out, nil
	}
	return nil, fmt.Errorf("preview extraction failed for %s: %w", path, err)
}
