package main

import (
	"database/sql"
	"log"
	"embed"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

//go:embed db/migrations/*.sql
var embedMigrations embed.FS

const (
	migrationsPath = "db/migrations"
)

type Store struct {
	db *sql.DB
}

func StoreInit(path string) *Store {
	db, err := Connect("file:" + path + "?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		log.Fatal(err)
	}

	return &Store{
		db: db,
	}
}

func (s *Store) Deinit() {
	s.db.Close()
}

func Connect(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return nil, err
	}

	if err := goose.Up(db, migrationsPath); err != nil {
		return nil, err
	}

	return db, nil
}
