package db

import (
	"context"
)

type Track struct {
	ID          int64  `db:"id"`
	Source      string `db:"source"`
	SourceID    string `db:"source_id"`
	Title       string `db:"title"`
	Artist      string `db:"artist"`
	Duration    int64  `db:"duration"`
	FilePath    string `db:"file_path"`
	CreatedAt   int64  `db:"created_at"`
	AlbumArtURL string `db:"album_art_url"`
	LastUsedAt  int64  `db:"last_used_at"`
}

func (d *DB) GetTrack(ctx context.Context, sourceID string) (*Track, error) {
	track := &Track{}
	err := d.conn.GetContext(ctx, track, "SELECT * from tracks WHERE source_id=$1", sourceID)
	if err != nil {
		return nil, err
	}
	return track, nil
}

func (d *DB) CreateTrack(ctx context.Context, source, sourceID, title, artist, filePath string, duration int64, albumArtURL string) (*Track, error) {
	_, err := d.conn.ExecContext(ctx, `
	INSERT INTO tracks
	(source, source_id, title, artist, duration, file_path, album_art_url, last_used_at)
	VALUES
	($1, $2, $3, $4, $5, $6, $7, unixepoch())`,
		source, sourceID, title, artist, duration, filePath, albumArtURL)
	if err != nil {
		return nil, err
	}
	return d.GetTrack(ctx, sourceID)
}

func (d *DB) UpdateTrackAlbumArtURL(ctx context.Context, trackID int64, albumArtURL string) error {
	_, err := d.conn.ExecContext(ctx, "UPDATE tracks SET album_art_url=$1 WHERE id=$2", albumArtURL, trackID)
	return err
}

func (d *DB) TouchTrackLastUsed(ctx context.Context, trackID int64) error {
	_, err := d.conn.ExecContext(ctx, "UPDATE tracks SET last_used_at=unixepoch() WHERE id=$1", trackID)
	return err
}

func (d *DB) ListTracksByLastUsed(ctx context.Context) ([]*Track, error) {
	tracks := []*Track{}
	err := d.conn.SelectContext(ctx, &tracks, "SELECT * FROM tracks ORDER BY last_used_at DESC")
	if err != nil {
		return nil, err
	}
	return tracks, nil
}
