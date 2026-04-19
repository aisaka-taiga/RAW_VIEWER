package rawconvert

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type ExifToolConverter struct {
	ExePath string
}

func DefaultExifToolConverter() ExifToolConverter {
	return ExifToolConverter{
		ExePath: `C:\workspace\exiftool-13.55_64\exiftool(-k).exe`,
	}
}

func (c ExifToolConverter) Convert(inputPath, outputPath string) error {
	if _, err := os.Stat(inputPath); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}

	data, err := c.extractPreview(inputPath)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("no embedded preview found in %s", inputPath)
	}

	return os.WriteFile(outputPath, data, 0o644)
}

func (c ExifToolConverter) extractPreview(inputPath string) ([]byte, error) {
	cmd := exec.Command(c.ExePath, "-q", "-q", "-b", "-JpgFromRaw", inputPath)
	out, err := cmd.Output()
	if err == nil && len(out) > 0 {
		return out, nil
	}

	cmd = exec.Command(c.ExePath, "-q", "-q", "-b", "-PreviewImage", inputPath)
	out, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("exiftool preview extraction failed: %w", err)
	}
	return out, nil
}

func Convert(inputPath, outputPath string) error {
	return DefaultExifToolConverter().Convert(inputPath, outputPath)
}
