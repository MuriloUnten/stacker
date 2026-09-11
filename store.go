package main

import (
	"database/sql"
	"log"
	"github.com/pressly/goose/v3"
	"embed"
)

//go:embed db/migrations/*.sql
var embedMigrations embed.FS

const (
	migrationsPath = "db/migrations"
)

type Store struct {
	db *sql.DB
}

func StoreInit() *Store {
	db, err := Connect("file:db/database.db?_foreign_keys=on")
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
