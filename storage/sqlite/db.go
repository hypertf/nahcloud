package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const defaultDSN = "file:nah.db?_busy_timeout=5000&_fk=1"

type DB struct{ *sql.DB }

func NewDB(dsn string) (*DB, error) {
	if dsn == "" {
		dsn = defaultDSN
	}
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	dsn += separator + "_foreign_keys=on&_busy_timeout=5000"
	database, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	for _, pragma := range []string{"PRAGMA foreign_keys = ON", "PRAGMA journal_mode = WAL", "PRAGMA busy_timeout = 5000"} {
		if _, err := database.Exec(pragma); err != nil {
			database.Close()
			return nil, fmt.Errorf("set %s: %w", pragma, err)
		}
	}
	db := &DB{DB: database}
	if err := db.runMigrations(); err != nil {
		database.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	return db, nil
}
