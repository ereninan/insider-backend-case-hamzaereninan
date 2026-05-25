package db

import (
	"database/sql"
	"errors"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func Open(dsn, schemaPath string) (*sql.DB, error) {
	database, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, errors.New("veritabanı açılamadı")
	}

	database.SetMaxOpenConns(1)

	err = database.Ping()
	if err != nil {
		return nil, errors.New("ping atılamadı")
	}

	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, errors.New("schema dosyası okunamadı")
	}

	_, err = database.Exec(string(schema))
	if err != nil {
		return nil, errors.New("schema çalıştırılamadı")
	}

	return database, nil
}
