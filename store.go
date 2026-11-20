package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS applications (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	company TEXT NOT NULL,
	reported TEXT NOT NULL DEFAULT 'Yes',
	role_title TEXT NOT NULL,
	location TEXT NOT NULL DEFAULT '',
	comp_range TEXT NOT NULL DEFAULT '',
	source TEXT NOT NULL DEFAULT '',
	date_reviewed TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'Applied',
	fit TEXT NOT NULL DEFAULT '',
	notes TEXT NOT NULL DEFAULT '',
	tailored_resume TEXT NOT NULL DEFAULT '',
	next_action TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

// defaultDBPath returns where the database file lives.
// JOBTRACKER_DB wins if set, otherwise the macOS app data folder.
func defaultDBPath() (string, error) {
	if p := os.Getenv("JOBTRACKER_DB"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "go-job-tracker", "jobtracker.db"), nil
}

// OpenStore opens the database, creating the folder, file and table if needed.
func OpenStore(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db folder: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}
	return db, nil
}
