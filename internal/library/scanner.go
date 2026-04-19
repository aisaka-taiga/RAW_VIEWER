package library

import (
	"io/fs"
	"path/filepath"
)

func Walk(root string, fn func(string) error) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !IsSupported(path) {
			return nil
		}
		return fn(path)
	})
}
