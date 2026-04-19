package thumbnail

import (
	"bytes"
	"context"
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

	entry1, data1, err := svc.Get(context.Background(), rawPath, 128)
	if err != nil {
		t.Fatal(err)
	}
	if entry1.FromDisk {
		t.Fatal("first read should not hit disk")
	}
	if len(data1) == 0 {
		t.Fatal("expected thumbnail bytes")
	}
	info, err := os.Stat(rawPath)
	if err != nil {
		t.Fatal(err)
	}
	key := svc.cache.Key(rawPath, 128, info.ModTime(), info.Size())
	if _, ok, err := svc.cache.Load(key); err != nil {
		t.Fatal(err)
	} else if !ok {
		t.Fatal("expected thumbnail to be persisted on disk")
	}

	entry2, data2, err := svc.Get(context.Background(), rawPath, 128)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data1, data2) {
		t.Fatal("cached bytes mismatch")
	}
	if entry2.Key != entry1.Key {
		t.Fatal("expected identical cache keys")
	}
}

type fakePreviewer struct {
	data  []byte
	calls int
}

func (p *fakePreviewer) GetPreviewBytes(ctx context.Context, path string) ([]byte, error) {
	p.calls++
	return p.data, nil
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

func TestWarmRawCachesPreviewAndThumbnail(t *testing.T) {
	dir := t.TempDir()
	rawPath := filepath.Join(dir, "warm.arw")
	if err := os.WriteFile(rawPath, []byte("raw"), 0o644); err != nil {
		t.Fatal(err)
	}

	src := image.NewRGBA(image.Rect(0, 0, 640, 480))
	src.Set(0, 0, color.RGBA{0, 255, 0, 255})
	var previewBuf bytes.Buffer
	if err := jpeg.Encode(&previewBuf, src, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}

	prev := &fakePreviewer{data: previewBuf.Bytes()}
	svc := &Service{
		cache:     NewDiskCache(dir),
		previewer: prev,
		memCache:  make(map[string]memEntry),
	}

	if err := svc.WarmRaw(context.Background(), rawPath, 128); err != nil {
		t.Fatal(err)
	}
	if prev.calls != 1 {
		t.Fatalf("expected one preview extraction, got %d", prev.calls)
	}
	info, err := os.Stat(rawPath)
	if err != nil {
		t.Fatal(err)
	}
	key := svc.cache.Key(rawPath, 128, info.ModTime(), info.Size())
	if _, ok, err := svc.cache.Load(key); err != nil {
		t.Fatal(err)
	} else if !ok {
		t.Fatal("expected warmed thumbnail to be persisted on disk")
	}

	entry, data, err := svc.Get(context.Background(), rawPath, 128)
	if err != nil {
		t.Fatal(err)
	}
	if prev.calls != 1 {
		t.Fatalf("expected cached preview to skip extra extraction, got %d calls", prev.calls)
	}
	if entry.Key != key {
		t.Fatalf("expected cache key %s, got %s", key, entry.Key)
	}
	if _, err := jpeg.Decode(bytes.NewReader(data)); err != nil {
		t.Fatalf("expected valid cached jpeg output: %v", err)
	}
}
