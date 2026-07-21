package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upInit, downInit)
}

func upInit(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE tracks (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	source     TEXT    NOT NULL,
	source_id  TEXT    NOT NULL,
	title      TEXT    NOT NULL,
	artist     TEXT    NOT NULL,
	duration   INTEGER NOT NULL,
	file_path  TEXT    NOT NULL,
	created_at INTEGER NOT NULL DEFAULT (unixepoch()),
	UNIQUE(source, source_id)
)
`)
	return err
}

func downInit(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`DROP TABLE tracks`)
	return err
}
