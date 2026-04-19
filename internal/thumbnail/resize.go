package thumbnail

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
)

func DecodeAndResize(path string, maxSize int, decoder Decoder) ([]byte, int, int, string, error) {
	img, mimeType, err := decoder.Decode(path, maxSize)
	if err != nil {
		return nil, 0, 0, "", err
	}
	resized := ResizeImage(img, maxSize)
	return encodeJPEG(resized, mimeType)
}

func encodeJPEG(img image.Image, mimeType ...string) ([]byte, int, int, string, error) {
	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, 0, 0, "", err
	}
	bounds := img.Bounds()
	outType := "image/jpeg"
	if len(mimeType) > 0 && mimeType[0] != "" {
		outType = mimeType[0]
	}
	return buf.Bytes(), bounds.Dx(), bounds.Dy(), outType, nil
}

func ResizeImage(src image.Image, maxSize int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxSize && h <= maxSize {
		return src
	}

	ratio := float64(w) / float64(h)
	var nw, nh int
	if w >= h {
		nw = maxSize
		nh = int(float64(maxSize) / ratio)
	} else {
		nh = maxSize
		nw = int(float64(maxSize) * ratio)
	}

	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		for x := 0; x < nw; x++ {
			sx := b.Min.X + x*w/nw
			sy := b.Min.Y + y*h/nh
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func ReadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}
