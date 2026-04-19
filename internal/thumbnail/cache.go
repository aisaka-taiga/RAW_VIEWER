package thumbnail

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type DiskCache struct {
	root string
	mu   sync.Mutex
}

func NewDiskCache(root string) *DiskCache {
	return &DiskCache{root: root}
}

func (c *DiskCache) Key(path string, size int, modTime time.Time, fileSize int64) string {
	return c.keyWithVariant(path, fmt.Sprintf("thumb:%d", size), modTime, fileSize)
}

func (c *DiskCache) RawPreviewKey(path string, size int, modTime time.Time, fileSize int64) string {
	return c.keyWithVariant(path, fmt.Sprintf("raw-preview:%d", size), modTime, fileSize)
}

func (c *DiskCache) keyWithVariant(path, variant string, modTime time.Time, fileSize int64) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s|%s|%d|%d|v6", path, variant, modTime.UnixNano(), fileSize)))
	return hex.EncodeToString(sum[:])
}

func (c *DiskCache) PathFor(key string) string {
	return filepath.Join(c.root, key+".jpg")
}

func (c *DiskCache) Load(key string) ([]byte, bool, error) {
	p := c.PathFor(key)
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return b, true, nil
}

func (c *DiskCache) Save(key string, data []byte) (string, error) {
	if err := os.MkdirAll(c.root, 0o755); err != nil {
		return "", err
	}
	p := c.PathFor(key)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return "", err
	}
	return p, nil
}
