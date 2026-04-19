package thumbnail

import "image"

type CacheEntry struct {
	Key      string
	Path     string
	FromDisk  bool
	Width    int
	Height   int
	MimeType string
}

type Decoder interface {
	Decode(path string, maxSize int) (image.Image, string, error)
}
