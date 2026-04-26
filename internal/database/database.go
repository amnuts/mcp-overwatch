package database

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func OpenCatalogue(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes
	if err := applyCatalogueMigrations(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func OpenStats(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := applyStatsMigrations(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
