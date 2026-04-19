package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB *sql.DB
}

type Photo struct {
	ID          string
	Path        string
	FileName    string
	FolderPath  string
	Format      string
	Width       int
	Height      int
	Orientation int
	CapturedAt  string
	ModifiedAt  time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Metadata struct {
	PhotoID        string
	Make           string
	Model          string
	ISO            string
	Aperture       string
	ShutterSpeed   string
	FocalLength    string
	LensModel      string
	RawJSON        string
}

func (s *Store) Init() error {
	stmts := []string{
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA synchronous = NORMAL;`,
		`PRAGMA busy_timeout = 5000;`,
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS photos (
			id TEXT PRIMARY KEY,
			path TEXT NOT NULL UNIQUE,
			file_name TEXT NOT NULL,
			folder_path TEXT NOT NULL,
			format TEXT NOT NULL,
			width INTEGER NOT NULL DEFAULT 0,
			height INTEGER NOT NULL DEFAULT 0,
			orientation INTEGER NOT NULL DEFAULT 1,
			captured_at TEXT NOT NULL DEFAULT '',
			modified_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS photo_metadata (
			photo_id TEXT PRIMARY KEY,
			make TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL DEFAULT '',
			iso TEXT NOT NULL DEFAULT '',
			aperture TEXT NOT NULL DEFAULT '',
			shutter_speed TEXT NOT NULL DEFAULT '',
			focal_length TEXT NOT NULL DEFAULT '',
			lens_model TEXT NOT NULL DEFAULT '',
			raw_json TEXT NOT NULL DEFAULT '',
			FOREIGN KEY(photo_id) REFERENCES photos(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS thumbnail_cache (
			photo_id TEXT PRIMARY KEY,
			cache_key TEXT NOT NULL,
			thumb_path TEXT NOT NULL,
			mime_type TEXT NOT NULL,
			width INTEGER NOT NULL,
			height INTEGER NOT NULL,
			created_at DATETIME NOT NULL,
			last_accessed_at DATETIME NOT NULL,
			FOREIGN KEY(photo_id) REFERENCES photos(id) ON DELETE CASCADE
		);`,
	}
	for _, stmt := range stmts {
		if _, err := s.DB.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UpsertPhoto(p Photo) error {
	_, err := s.DB.Exec(`INSERT INTO photos(id, path, file_name, folder_path, format, width, height, orientation, captured_at, modified_at, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			path=excluded.path,
			file_name=excluded.file_name,
			folder_path=excluded.folder_path,
			format=excluded.format,
			width=excluded.width,
			height=excluded.height,
			orientation=excluded.orientation,
			captured_at=excluded.captured_at,
			modified_at=excluded.modified_at,
			updated_at=excluded.updated_at`,
		p.ID, p.Path, p.FileName, p.FolderPath, p.Format, p.Width, p.Height, p.Orientation, p.CapturedAt, p.ModifiedAt, p.CreatedAt, p.UpdatedAt)
	return err
}

func (s *Store) UpsertMetadata(m Metadata) error {
	_, err := s.DB.Exec(`INSERT INTO photo_metadata(photo_id, make, model, iso, aperture, shutter_speed, focal_length, lens_model, raw_json)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(photo_id) DO UPDATE SET
			make=excluded.make,
			model=excluded.model,
			iso=excluded.iso,
			aperture=excluded.aperture,
			shutter_speed=excluded.shutter_speed,
			focal_length=excluded.focal_length,
			lens_model=excluded.lens_model,
			raw_json=excluded.raw_json`,
		m.PhotoID, m.Make, m.Model, m.ISO, m.Aperture, m.ShutterSpeed, m.FocalLength, m.LensModel, m.RawJSON)
	return err
}

func (s *Store) GetMetadata(photoID string) (Metadata, error) {
	row := s.DB.QueryRow(`SELECT photo_id, make, model, iso, aperture, shutter_speed, focal_length, lens_model, raw_json
		FROM photo_metadata WHERE photo_id = ?`, photoID)

	var m Metadata
	if err := row.Scan(&m.PhotoID, &m.Make, &m.Model, &m.ISO, &m.Aperture, &m.ShutterSpeed, &m.FocalLength, &m.LensModel, &m.RawJSON); err != nil {
		return Metadata{}, err
	}
	return m, nil
}

func (s *Store) GetPhoto(photoID string) (Photo, error) {
	row := s.DB.QueryRow(`SELECT id, path, file_name, folder_path, format, width, height, orientation, captured_at, modified_at, created_at, updated_at
		FROM photos WHERE id = ?`, photoID)

	var p Photo
	if err := row.Scan(&p.ID, &p.Path, &p.FileName, &p.FolderPath, &p.Format, &p.Width, &p.Height, &p.Orientation, &p.CapturedAt, &p.ModifiedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return Photo{}, err
	}
	return p, nil
}

func (s *Store) DeletePhoto(photoID string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM photo_metadata WHERE photo_id = ?`, photoID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM thumbnail_cache WHERE photo_id = ?`, photoID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM photos WHERE id = ?`, photoID); err != nil {
		return err
	}
	return tx.Commit()
}

func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func (s *Store) ListPhotos(folderPath string, limit, offset uint32, sortBy string, descending bool) ([]Photo, uint64, error) {
	orderBy := "modified_at"
	switch sortBy {
	case "file_name":
		orderBy = "file_name"
	case "format":
		orderBy = "format"
	case "captured_at":
		orderBy = "captured_at"
	case "modified_at":
		orderBy = "modified_at"
	case "":
		orderBy = "modified_at"
	default:
		return nil, 0, fmt.Errorf("unsupported sort field: %s", sortBy)
	}

	orderDir := "ASC"
	if descending {
		orderDir = "DESC"
	}

	args := []any{}
	where := ""
	if folderPath != "" {
		where = "WHERE folder_path = ?"
		args = append(args, folderPath)
	}

	var total uint64
	countSQL := "SELECT COUNT(*) FROM photos " + where
	if err := s.DB.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`SELECT id, path, file_name, folder_path, format, width, height, orientation, captured_at, modified_at, created_at, updated_at
		FROM photos %s ORDER BY %s %s LIMIT ? OFFSET ?`, where, orderBy, orderDir)
	args = append(args, limit, offset)

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Photo
	for rows.Next() {
		var p Photo
		if err := rows.Scan(&p.ID, &p.Path, &p.FileName, &p.FolderPath, &p.Format, &p.Width, &p.Height, &p.Orientation, &p.CapturedAt, &p.ModifiedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}
