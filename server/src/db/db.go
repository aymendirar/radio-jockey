package db

import (
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
	_ "server/src/db/migrations"
)

const (
	// confusingly, these identifiers are slightly different
	GooseDBIdentifier = "sqlite3"
	SQLXDBIdentifier  = "sqlite"
)

type DB struct {
	conn *sqlx.DB
}

func Open(path string) (*DB, error) {
	goose.SetDialect(GooseDBIdentifier)

	conn, err := sqlx.Connect(SQLXDBIdentifier, path)
	if err != nil {
		return nil, err
	}

	// this looks for a migrations/ directory from the current . directory
	if err := goose.Up(conn.DB, "."); err != nil {
		return nil, err
	}

	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}
