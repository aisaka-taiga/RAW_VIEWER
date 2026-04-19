package indexer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"photo-viewer/internal/db"
)

type Indexer struct {
	Store *db.Store
	Meta  interface {
		Read(string) (map[string]string, string, error)
	}
}

func (i Indexer) IndexFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	fields := map[string]string{}
	rawJSON := ""
	if i.Meta != nil {
		if metaFields, metaRawJSON, err := i.Meta.Read(path); err == nil {
			fields = metaFields
			rawJSON = metaRawJSON
		}
	}

	now := time.Now().UTC()
	photo := db.Photo{
		ID:          path,
		Path:        path,
		FileName:    filepath.Base(path),
		FolderPath:  filepath.Dir(path),
		Format:      filepath.Ext(path),
		Width:       parseInt(fields["ImageWidth"]),
		Height:      parseInt(fields["ImageHeight"]),
		Orientation: parseOrientation(fields["Orientation"]),
		CapturedAt:  fields["DateTimeOriginal"],
		ModifiedAt:  info.ModTime().UTC(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := i.Store.UpsertPhoto(photo); err != nil {
		return err
	}

	meta := db.Metadata{
		PhotoID:      path,
		Make:         fields["Make"],
		Model:        fields["Model"],
		ISO:          fields["ISO"],
		Aperture:     firstNonEmpty(fields["Aperture"], fields["FNumber"]),
		ShutterSpeed: fields["ShutterSpeed"],
		FocalLength:  fields["FocalLength"],
		LensModel:    fields["LensModel"],
		RawJSON:      rawJSON,
	}
	return i.Store.UpsertMetadata(meta)
}

func parseInt(s string) int {
	var v int
	fmt.Sscanf(s, "%d", &v)
	return v
}

func parseOrientation(s string) int {
	var v int
	if fmt.Sscanf(s, "%d", &v); v > 0 {
		return v
	}
	// Handle string orientations if needed, but ExifTool -n gives numbers
	return 1
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func OpenSQLite(path string) (*sql.DB, error) {
	// Enable WAL mode and busy timeout to prevent "database is locked" errors
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}
