package metadata

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type ExifToolReader struct {
	ExePath string
}

func DefaultExifToolReader() ExifToolReader {
	return ExifToolReader{ExePath: `C:\workspace\exiftool-13.55_64\exiftool(-k).exe`}
}

type exifJSON map[string]any

func (r ExifToolReader) Read(path string) (map[string]string, string, error) {
	cmd := exec.Command(r.ExePath, "-j", "-n", path)
	out, err := cmd.Output()
	if err != nil {
		return nil, "", fmt.Errorf("exiftool metadata failed: %w", err)
	}
	var arr []exifJSON
	if err := json.Unmarshal(out, &arr); err != nil {
		return nil, "", err
	}
	if len(arr) == 0 {
		return nil, "", fmt.Errorf("no metadata returned")
	}
	m := make(map[string]string)
	for k, v := range arr[0] {
		switch val := v.(type) {
		case string:
			m[k] = val
		case float64:
			m[k] = fmt.Sprintf("%v", val)
		case bool:
			m[k] = fmt.Sprintf("%t", val)
		}
	}
	return m, string(out), nil
}
