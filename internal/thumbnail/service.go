package thumbnail

import (
	"bytes"
	"context"
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
		GetPreviewBytes(context.Context, string) ([]byte, error)
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
			if ori, ok := readOrientation(path, s.exifToolPath); ok {
				img = applyOrientation(img, ori)
			}
			data, width, height, _, err = encodeThumbnail(img, size)
			mimeType = "image/jpeg"
		} else if s.decoder != nil {
			data, width, height, mimeType, err = DecodeAndResize(path, size, s.decoder)
		}
	} else if s.decoder != nil {
		data, width, height, mimeType, err = DecodeAndResize(path, size, s.decoder)
	} else {
		data, err = s.previewer.GetPreviewBytes(ctx, path)
		if err == nil && len(data) > 0 {
			// Optimization: Check orientation FIRST from DB or ExifTool
			ori := 1
			if s.store != nil {
				if p, err := s.store.GetPhoto(path); err == nil {
					ori = p.Orientation
				}
			} else {
				if o, ok := readOrientation(path, s.exifToolPath); ok {
					ori = o
				}
			}

			if ori <= 1 {
				// Super Fast Path: Orientation is normal (1), set values and move to cache save
				mimeType = "image/jpeg"
				width, height = 0, 0 // Client handles display
			} else {
				// If orientation is not 1, we must decode it to apply rotation
				if img, _, decodeErr := image.Decode(bytes.NewReader(data)); decodeErr == nil && img != nil {
					if ori > 1 {
						img = applyOrientation(img, ori)
						data, width, height, _, err = encodeThumbnail(img, size)
					} else {
						b := img.Bounds()
						width, height = b.Dx(), b.Dy()
					}
					mimeType = "image/jpeg"
				} else {
					width, height = 0, 0
					mimeType = http.DetectContentType(data)
				}
			}
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
