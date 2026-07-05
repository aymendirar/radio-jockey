package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddTrackAlbumArt, downAddTrackAlbumArt)
}

func upAddTrackAlbumArt(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE tracks ADD COLUMN album_art_url TEXT`); err != nil {
		return err
	}
	_, err := tx.Exec(`
UPDATE tracks SET album_art_url = 'https://i.ytimg.com/vi/' || source_id || '/hqdefault.jpg'
WHERE source = 'youtube' AND album_art_url IS NULL
`)
	return err
}

func downAddTrackAlbumArt(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE tracks DROP COLUMN album_art_url`)
	return err
}
