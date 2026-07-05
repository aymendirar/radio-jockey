package db

import (
	"context"
)

type SessionArchive struct {
	Id        int64  `db:"id"`
	SessionId string `db:"session_id"`
	CreatedAt int64  `db:"created_at"`
}

func (d *DB) CreateSessionArchive(ctx context.Context, sessionId string) (*SessionArchive, error) {
	res, err := d.conn.ExecContext(ctx, `
	INSERT INTO session_archives (session_id) VALUES ($1)`, sessionId)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return d.GetSessionArchive(ctx, id)
}

func (d *DB) GetSessionArchive(ctx context.Context, id int64) (*SessionArchive, error) {
	archive := &SessionArchive{}
	err := d.conn.GetContext(ctx, archive, "SELECT * FROM session_archives WHERE id=$1", id)
	if err != nil {
		return nil, err
	}
	return archive, nil
}

func (d *DB) ListSessionArchives(ctx context.Context) ([]*SessionArchive, error) {
	archives := []*SessionArchive{}
	err := d.conn.SelectContext(ctx, &archives, "SELECT * FROM session_archives ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	return archives, nil
}

func (d *DB) AddSessionArchiveTrack(ctx context.Context, archiveId, trackId int64) error {
	_, err := d.conn.ExecContext(ctx, `
	INSERT INTO session_archive_tracks (session_archive_id, track_id) VALUES ($1, $2)`, archiveId, trackId)
	return err
}

func (d *DB) ListSessionArchiveTracks(ctx context.Context, archiveId int64) ([]*Track, error) {
	tracks := []*Track{}
	err := d.conn.SelectContext(ctx, &tracks, `
	SELECT tracks.* FROM tracks
	JOIN session_archive_tracks ON session_archive_tracks.track_id = tracks.id
	WHERE session_archive_tracks.session_archive_id = $1
	ORDER BY session_archive_tracks.played_at ASC`, archiveId)
	if err != nil {
		return nil, err
	}
	return tracks, nil
}

func (d *DB) DeleteSessionArchive(ctx context.Context, id int64) error {
	if _, err := d.conn.ExecContext(ctx, "DELETE FROM session_archive_tracks WHERE session_archive_id=$1", id); err != nil {
		return err
	}
	_, err := d.conn.ExecContext(ctx, "DELETE FROM session_archives WHERE id=$1", id)
	return err
}
