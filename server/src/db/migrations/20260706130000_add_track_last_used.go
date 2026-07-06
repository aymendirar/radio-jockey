package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddTrackLastUsed, downAddTrackLastUsed)
}

func upAddTrackLastUsed(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE tracks ADD COLUMN last_used_at INTEGER`); err != nil {
		return err
	}
	_, err := tx.Exec(`UPDATE tracks SET last_used_at = created_at WHERE last_used_at IS NULL`)
	return err
}

func downAddTrackLastUsed(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE tracks DROP COLUMN last_used_at`)
	return err
}
