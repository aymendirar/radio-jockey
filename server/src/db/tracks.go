package db

import (
	"context"
)

type TracksDB interface {
	CreateTrack(ctx context.Context, source, sourceId, title, artist, filePath string, duration int64, albumArtUrl string) (*Track, error)
	GetTrack(ctx context.Context, trackId string) (*Track, error)
	UpdateTrackAlbumArtUrl(ctx context.Context, trackId int64, albumArtUrl string) error
	TouchTrackLastUsed(ctx context.Context, trackId int64) error
	ListTracksByLastUsed(ctx context.Context) ([]*Track, error)
}

type Track struct {
	Id          int64  `db:"id"`
	Source      string `db:"source"`
	SourceId    string `db:"source_id"`
	Title       string `db:"title"`
	Artist      string `db:"artist"`
	Duration    int64  `db:"duration"`
	FilePath    string `db:"file_path"`
	CreatedAt   int64  `db:"created_at"`
	AlbumArtUrl string `db:"album_art_url"`
	LastUsedAt  int64  `db:"last_used_at"`
}

func (d *DB) GetTrack(ctx context.Context, sourceId string) (*Track, error) {
	track := &Track{}
	err := d.conn.GetContext(ctx, track, "SELECT * from tracks WHERE source_id=$1", sourceId)
	if err != nil {
		return nil, err
	}
	return track, nil
}

func (d *DB) CreateTrack(ctx context.Context, source, sourceId, title, artist, filePath string, duration int64, albumArtUrl string) (*Track, error) {
	_, err := d.conn.ExecContext(ctx, `
	INSERT INTO tracks
	(source, source_id, title, artist, duration, file_path, album_art_url, last_used_at)
	VALUES
	($1, $2, $3, $4, $5, $6, $7, unixepoch())`,
		source, sourceId, title, artist, duration, filePath, albumArtUrl)
	if err != nil {
		return nil, err
	}
	return d.GetTrack(ctx, sourceId)
}

func (d *DB) UpdateTrackAlbumArtUrl(ctx context.Context, trackId int64, albumArtUrl string) error {
	_, err := d.conn.ExecContext(ctx, "UPDATE tracks SET album_art_url=$1 WHERE id=$2", albumArtUrl, trackId)
	return err
}

func (d *DB) TouchTrackLastUsed(ctx context.Context, trackId int64) error {
	_, err := d.conn.ExecContext(ctx, "UPDATE tracks SET last_used_at=unixepoch() WHERE id=$1", trackId)
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
