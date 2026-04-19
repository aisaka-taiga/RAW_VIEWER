package thumbnail

import (
	"image"
	"os/exec"
	"strconv"
	"strings"
)

func readOrientation(path, exifToolPath string) (int, bool) {
	if exifToolPath == "" {
		exifToolPath = "exiftool"
	}
	cmd := exec.Command(exifToolPath, "-q", "-q", "-Orientation#", "-S", path)
	out, err := cmd.Output()
	if err != nil {
		return 0, false
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return 0, false
	}
	// Take the first numerical value if multiple are returned (e.g. "Orientation: 6")
	parts := strings.Fields(text)
	valStr := ""
	if len(parts) > 1 {
		valStr = parts[len(parts)-1] // Usually ExifTool -S returns "Tag: Val"
	} else if len(parts) == 1 {
		valStr = parts[0]
	}
	
	if valStr == "" {
		return 0, false
	}

	n, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, false
	}
	return n, true
}

func applyOrientation(img image.Image, orientation int) image.Image {
	switch orientation {
	case 3:
		return rotate180(img)
	case 6:
		return rotate90CW(img)
	case 8:
		return rotate90CCW(img)
	default:
		return img
	}
}

func rotate90CW(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.Set(b.Dy()-1-y, x, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func rotate90CCW(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.Set(y, b.Dx()-1-x, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func rotate180(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.Set(b.Dx()-1-x, b.Dy()-1-y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
