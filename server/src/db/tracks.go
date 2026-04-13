package db

import (
	"context"
)

type TracksDB interface {
	CreateTrack(ctx context.Context, track Track) (*Track, error)
	GetTrack(ctx context.Context, trackId string) (*Track, error)
}

type Track struct {
	Id        int64  `db:"id"`
	Source    string `db:"source"`
	SourceId  string `db:"source"`
	Title     string `db:"source"`
	Artist    string `db:"source"`
	Duration  int64  `db:"source"`
	FilePath  int64  `db:"source"`
	CreatedAt int64  `db:"source"`
}

func (d *DB) GetTrack(ctx context.Context, sourceId string) {

}

func (d *DB) CreateTrack(ctx context.Context, query Track) {

}
