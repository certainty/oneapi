package storage

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

// DB represents a database connection
type DB struct {
	*sql.DB
}

func NewSQLiteDB() (*DB, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

func (db *DB) Exec(query string, args ...any) error {
	_, err := db.DB.Exec(query, args...)
	return err
}
