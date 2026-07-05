package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddSessionArchives, downAddSessionArchives)
}

func upAddSessionArchives(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE session_archives (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT    NOT NULL,
	created_at INTEGER NOT NULL DEFAULT (unixepoch())
)
`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
CREATE TABLE session_archive_tracks (
	id                 INTEGER PRIMARY KEY AUTOINCREMENT,
	session_archive_id INTEGER NOT NULL REFERENCES session_archives(id),
	track_id           INTEGER NOT NULL REFERENCES tracks(id),
	played_at          INTEGER NOT NULL DEFAULT (unixepoch())
)
`)
	return err
}

func downAddSessionArchives(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.Exec(`DROP TABLE session_archive_tracks`); err != nil {
		return err
	}
	_, err := tx.Exec(`DROP TABLE session_archives`)
	return err
}
