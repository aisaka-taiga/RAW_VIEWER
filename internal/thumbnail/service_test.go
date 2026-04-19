package thumbnail

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestIsRaw(t *testing.T) {
	if !IsRaw("a.ARW") || !IsRaw("b.cr3") || !IsRaw("c.nef") {
		t.Fatal("expected raw extensions to be detected")
	}
	if IsRaw("d.jpg") {
		t.Fatal("jpg should not be raw")
	}
}

func TestCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(dir, FakeRawDecoder{})

	rawPath := filepath.Join(dir, "sample.arw")
	if err := os.WriteFile(rawPath, []byte("raw"), 0o644); err != nil {
		t.Fatal(err)
	}

	entry1, data1, err := svc.Get(rawPath, 128)
	if err != nil {
		t.Fatal(err)
	}
	if entry1.FromDisk {
		t.Fatal("first read should not hit disk")
	}
	if len(data1) == 0 {
		t.Fatal("expected thumbnail bytes")
	}

	entry2, data2, err := svc.Get(rawPath, 128)
	if err != nil {
		t.Fatal(err)
	}
	if !entry2.FromDisk {
		t.Fatal("second read should hit disk")
	}
	if !bytes.Equal(data1, data2) {
		t.Fatal("cached bytes mismatch")
	}
}

func TestResizeImage(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 800, 400))
	src.Set(0, 0, color.RGBA{255, 0, 0, 255})
	resized := ResizeImage(src, 200)
	if resized.Bounds().Dx() != 200 {
		t.Fatalf("expected width 200, got %d", resized.Bounds().Dx())
	}
	if resized.Bounds().Dy() != 100 {
		t.Fatalf("expected height 100, got %d", resized.Bounds().Dy())
	}
}

func TestDecodeAndResizeProducesJpeg(t *testing.T) {
	dir := t.TempDir()
	rawPath := filepath.Join(dir, "x.nef")
	if err := os.WriteFile(rawPath, []byte("raw"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, _, _, mimeType, err := DecodeAndResize(rawPath, 64, FakeRawDecoder{})
	if err != nil {
		t.Fatal(err)
	}
	if mimeType != "image/jpeg" {
		t.Fatalf("expected jpeg mime, got %s", mimeType)
	}
	if _, err := jpeg.Decode(bytes.NewReader(data)); err != nil {
		t.Fatalf("expected valid jpeg output: %v", err)
	}
}
