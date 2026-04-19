package thumbnail

import (
	"fmt"
	"image"
	"image/color"
	"strings"
)

type FakeRawDecoder struct{}

func (d FakeRawDecoder) Decode(path string, maxSize int) (image.Image, string, error) {
	ext := strings.ToLower(path)
	if !(strings.HasSuffix(ext, ".arw") || strings.HasSuffix(ext, ".cr3") || strings.HasSuffix(ext, ".nef")) {
		return nil, "", fmt.Errorf("unsupported raw format: %s", path)
	}

	size := maxSize
	if size <= 0 {
		size = 256
	}

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 160, A: 255})
		}
	}
	return img, "image/jpeg", nil
}
