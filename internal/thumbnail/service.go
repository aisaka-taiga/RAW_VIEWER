package thumbnail

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"photo-viewer/internal/db"
	"strings"
	"sync"
	"time"
)

type memEntry struct {
	data  []byte
	entry CacheEntry
}

type Service struct {
	cache        *DiskCache
	store        *db.Store
	decoder      Decoder
	exifToolPath string
	previewer    interface {
		GetPreviewBytes(context.Context, string, int) ([]byte, error)
	}

	// In-memory cache
	memCache map[string]memEntry
	memKeys  []string
	memLock  sync.Mutex
}

func NewService(root string, decoder Decoder) *Service {
	return &Service{
		cache:        NewDiskCache(root),
		decoder:      decoder,
		previewer:    DefaultExifToolPreviewer(),
		exifToolPath: DefaultExifToolPreviewer().ExePath,
		memCache:     make(map[string]memEntry),
	}
}

func NewExifToolService(root string) *Service {
	prev := DefaultExifToolPreviewer()
	return &Service{
		cache:        NewDiskCache(root),
		previewer:    prev,
		exifToolPath: prev.ExePath,
		memCache:     make(map[string]memEntry),
	}
}

func NewConfiguredService(root, exifToolPath string, store *db.Store) *Service {
	return &Service{
		cache:        NewDiskCache(root),
		store:        store,
		previewer:    Previewer{ExifToolPath: exifToolPath},
		exifToolPath: exifToolPath,
		memCache:     make(map[string]memEntry),
	}
}

func (s *Service) Get(ctx context.Context, path string, size int) (CacheEntry, []byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return CacheEntry{}, nil, err
	}
	key := s.cache.Key(path, size, info.ModTime(), info.Size())

	// 1. Check Memory Cache
	s.memLock.Lock()
	if m, ok := s.memCache[key]; ok {
		s.memLock.Unlock()
		m.entry.FromDisk = true
		return m.entry, m.data, nil
	}
	s.memLock.Unlock()

	// 2. Check Disk Cache
	if data, ok, err := s.cache.Load(key); err != nil {
		return CacheEntry{}, nil, err
	} else if ok {
		entry := CacheEntry{
			Key: key, Path: s.cache.PathFor(key), FromDisk: true, MimeType: "image/jpeg",
		}
		s.addToMem(key, entry, data)
		return entry, data, nil
	}

	var data []byte
	var width, height int
	var mimeType string

	if !IsRaw(path) {
		img, err := ReadImage(path)
		if err == nil && img != nil {
			if ori := s.resolveOrientation(path); ori > 1 {
				img = applyOrientation(img, ori)
			}
			data, width, height, _, err = encodeThumbnail(img, size)
			mimeType = "image/jpeg"
		} else if s.decoder != nil {
			data, width, height, mimeType, err = DecodeAndResize(path, size, s.decoder)
		}
	} else {
		previewBytes, previewErr := s.loadRawPreviewBytes(ctx, path, info, size)
		if previewErr == nil && len(previewBytes) > 0 {
			data, width, height, mimeType, err = s.renderThumbnailFromPreviewBytes(path, size, previewBytes)
			if err != nil && s.decoder != nil {
				data, width, height, mimeType, err = DecodeAndResize(path, size, s.decoder)
			}
		} else if s.decoder != nil {
			data, width, height, mimeType, err = DecodeAndResize(path, size, s.decoder)
		} else if previewErr != nil {
			err = previewErr
		} else {
			err = fmt.Errorf("empty raw preview for %s", path)
		}
	}
	if err != nil {
		return CacheEntry{}, nil, err
	}
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	saved, err := s.cache.Save(key, data)
	if err != nil {
		return CacheEntry{}, nil, err
	}
	entry := CacheEntry{
		Key: key, Path: saved, FromDisk: false, Width: width, Height: height, MimeType: mimeType,
	}
	s.addToMem(key, entry, data)
	return entry, data, nil
}

func (s *Service) WarmRaw(ctx context.Context, path string, sizes ...int) error {
	if !IsRaw(path) {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	for _, size := range normalizeWarmSizes(sizes...) {
		previewBytes, err := s.loadRawPreviewBytes(ctx, path, info, size)
		if err != nil {
			return err
		}
		if len(previewBytes) == 0 {
			return fmt.Errorf("empty raw preview for %s", path)
		}

		key := s.cache.Key(path, size, info.ModTime(), info.Size())
		if _, ok, err := s.cache.Load(key); err != nil {
			return err
		} else if ok {
			continue
		}

		data, width, height, mimeType, err := s.renderThumbnailFromPreviewBytes(path, size, previewBytes)
		if err != nil {
			return err
		}
		saved, err := s.cache.Save(key, data)
		if err != nil {
			return err
		}
		entry := CacheEntry{
			Key: key, Path: saved, FromDisk: false, Width: width, Height: height, MimeType: mimeType,
		}
		s.addToMem(key, entry, data)
	}

	return nil
}

func (s *Service) loadRawPreviewBytes(ctx context.Context, path string, info os.FileInfo, size int) ([]byte, error) {
	if s.previewer == nil {
		return nil, fmt.Errorf("raw previewer unavailable for %s", path)
	}
	previewKey := s.cache.RawPreviewKey(path, size, info.ModTime(), info.Size())
	if data, ok, err := s.cache.Load(previewKey); err != nil {
		return nil, err
	} else if ok {
		return data, nil
	}

	data, err := s.previewer.GetPreviewBytes(ctx, path, size)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty raw preview for %s", path)
	}
	if _, err := s.cache.Save(previewKey, data); err != nil {
		return nil, err
	}
	return data, nil
}

func (s *Service) renderThumbnailFromPreviewBytes(path string, size int, previewBytes []byte) ([]byte, int, int, string, error) {
	img, _, decodeErr := image.Decode(bytes.NewReader(previewBytes))
	if decodeErr != nil {
		return nil, 0, 0, "", decodeErr
	}

	if ori := s.resolveOrientation(path); ori > 1 {
		img = applyOrientation(img, ori)
	}

	data, width, height, mimeType, err := encodeThumbnail(img, size)
	if err != nil {
		return nil, 0, 0, "", err
	}
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	return data, width, height, mimeType, nil
}

func (s *Service) resolveOrientation(path string) int {
	if s.store != nil {
		if p, err := s.store.GetPhoto(path); err == nil && p.Orientation > 0 {
			return p.Orientation
		}
	}
	if o, ok := readOrientation(path, s.exifToolPath); ok {
		return o
	}
	return 1
}

func normalizeWarmSizes(sizes ...int) []int {
	if len(sizes) == 0 {
		return []int{384, 1024, 4096}
	}
	seen := make(map[int]struct{}, len(sizes))
	out := make([]int, 0, len(sizes))
	for _, size := range sizes {
		if size <= 0 {
			continue
		}
		if _, ok := seen[size]; ok {
			continue
		}
		seen[size] = struct{}{}
		out = append(out, size)
	}
	if len(out) == 0 {
		return []int{384, 1024, 4096}
	}
	return out
}

func (s *Service) addToMem(key string, entry CacheEntry, data []byte) {
	s.memLock.Lock()
	defer s.memLock.Unlock()

	// If already exists, just update data (unlikely to change with same key)
	if _, ok := s.memCache[key]; ok {
		s.memCache[key] = memEntry{data: data, entry: entry}
		return
	}

	// LRU eviction (max 1000 items)
	if len(s.memKeys) >= 1000 {
		oldest := s.memKeys[0]
		delete(s.memCache, oldest)
		s.memKeys = s.memKeys[1:]
	}

	s.memCache[key] = memEntry{data: data, entry: entry}
	s.memKeys = append(s.memKeys, key)
}

func encodeThumbnail(img image.Image, maxSize int) ([]byte, int, int, string, error) {
	resized := ResizeImage(img, maxSize)
	return encodeJPEG(resized)
}

func IsRaw(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".arw", ".cr3", ".nef", ".dng", ".rw2", ".heic", ".heif":
		return true
	default:
		return false
	}
}

func Touch(path string) error {
	now := time.Now().Local()
	return os.Chtimes(path, now, now)
}
