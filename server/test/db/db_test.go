package db_test

import (
	"context"
	"database/sql"
	"errors"
	"server/test/util"
	"testing"
)

func TestCreateTrack(t *testing.T) {
	d := util.OpenTestDB(t)

	track, err := d.CreateTrack(context.Background(), "youtube", "abc123", "Title", "Artist", "/music/abc123.opus", 180, "https://i.ytimg.com/vi/abc123/hqdefault.jpg")
	if err != nil {
		t.Fatalf("create track: %v", err)
	}
	if track.Id == 0 || track.Title != "Title" || track.Artist != "Artist" {
		t.Fatalf("unexpected track: %+v", track)
	}
	if track.AlbumArtUrl != "https://i.ytimg.com/vi/abc123/hqdefault.jpg" {
		t.Fatalf("unexpected album art url: %q", track.AlbumArtUrl)
	}
}

func TestUpdateTrackAlbumArtUrl(t *testing.T) {
	ctx := context.Background()
	d := util.OpenTestDB(t)

	track, err := d.CreateTrack(ctx, "youtube", "abc123", "Title", "Artist", "/music/abc123.opus", 180, "https://i.ytimg.com/vi/abc123/hqdefault.jpg")
	if err != nil {
		t.Fatalf("create track: %v", err)
	}

	if err := d.UpdateTrackAlbumArtUrl(ctx, track.Id, "https://i.ytimg.com/vi/abc123/maxresdefault.jpg"); err != nil {
		t.Fatalf("update album art url: %v", err)
	}

	got, err := d.GetTrack(ctx, "abc123")
	if err != nil {
		t.Fatalf("get track: %v", err)
	}
	if got.AlbumArtUrl != "https://i.ytimg.com/vi/abc123/maxresdefault.jpg" {
		t.Fatalf("expected updated album art url, got %q", got.AlbumArtUrl)
	}
}

func TestCreateTrackDuplicate(t *testing.T) {
	ctx := context.Background()
	d := util.OpenTestDB(t)

	if _, err := d.CreateTrack(ctx, "youtube", "abc123", "Title", "Artist", "/music/abc123.opus", 180, "https://i.ytimg.com/vi/abc123/hqdefault.jpg"); err != nil {
		t.Fatalf("create track: %v", err)
	}
	if _, err := d.CreateTrack(ctx, "youtube", "abc123", "Other Title", "Other Artist", "/music/other.opus", 200, "https://i.ytimg.com/vi/other/hqdefault.jpg"); err == nil {
		t.Fatal("expected error creating duplicate (source, source_id) track, got nil")
	}
}

func TestGetTrack(t *testing.T) {
	ctx := context.Background()
	d := util.OpenTestDB(t)

	created, err := d.CreateTrack(ctx, "youtube", "abc123", "Title", "Artist", "/music/abc123.opus", 180, "https://i.ytimg.com/vi/abc123/hqdefault.jpg")
	if err != nil {
		t.Fatalf("create track: %v", err)
	}

	got, err := d.GetTrack(ctx, "abc123")
	if err != nil {
		t.Fatalf("get track: %v", err)
	}
	if got.Id != created.Id || got.Title != "Title" || got.Artist != "Artist" {
		t.Fatalf("unexpected track: %+v", got)
	}
}

func TestGetTrackNotFound(t *testing.T) {
	d := util.OpenTestDB(t)

	if _, err := d.GetTrack(context.Background(), "nonexistent"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestCreateSessionArchive(t *testing.T) {
	archive, err := util.OpenTestDB(t).CreateSessionArchive(context.Background(), "test-session")
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	if archive.Id == 0 || archive.SessionId != "test-session" {
		t.Fatalf("unexpected archive: %+v", archive)
	}
}

func TestGetSessionArchiveNotFound(t *testing.T) {
	d := util.OpenTestDB(t)

	if _, err := d.GetSessionArchive(context.Background(), 999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListSessionArchives(t *testing.T) {
	ctx := context.Background()
	d := util.OpenTestDB(t)

	d.CreateSessionArchive(ctx, "session-a")
	d.CreateSessionArchive(ctx, "session-b")

	archives, err := d.ListSessionArchives(ctx)
	if err != nil {
		t.Fatalf("list archives: %v", err)
	}
	if len(archives) != 2 {
		t.Fatalf("expected 2 archives, got %v", archives)
	}
}

func TestAddAndListSessionArchiveTracks(t *testing.T) {
	ctx := context.Background()
	d := util.OpenTestDB(t)

	track, err := d.CreateTrack(ctx, "youtube", "abc123", "Title", "Artist", "/music/abc123.opus", 180, "https://i.ytimg.com/vi/abc123/hqdefault.jpg")
	if err != nil {
		t.Fatalf("create track: %v", err)
	}
	archive, err := d.CreateSessionArchive(ctx, "test-session")
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}

	if err := d.AddSessionArchiveTrack(ctx, archive.Id, track.Id); err != nil {
		t.Fatalf("add archive track: %v", err)
	}

	tracks, err := d.ListSessionArchiveTracks(ctx, archive.Id)
	if err != nil {
		t.Fatalf("list archive tracks: %v", err)
	}
	if len(tracks) != 1 || tracks[0].Id != track.Id {
		t.Fatalf("unexpected archive tracks: %v", tracks)
	}
}

func TestDeleteSessionArchive(t *testing.T) {
	ctx := context.Background()
	d := util.OpenTestDB(t)

	track, err := d.CreateTrack(ctx, "youtube", "abc123", "Title", "Artist", "/music/abc123.opus", 180, "https://i.ytimg.com/vi/abc123/hqdefault.jpg")
	if err != nil {
		t.Fatalf("create track: %v", err)
	}
	archive, err := d.CreateSessionArchive(ctx, "test-session")
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	if err := d.AddSessionArchiveTrack(ctx, archive.Id, track.Id); err != nil {
		t.Fatalf("add archive track: %v", err)
	}

	if err := d.DeleteSessionArchive(ctx, archive.Id); err != nil {
		t.Fatalf("delete archive: %v", err)
	}

	if _, err := d.GetSessionArchive(ctx, archive.Id); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows after delete, got %v", err)
	}
	if remaining, err := d.ListSessionArchiveTracks(ctx, archive.Id); err != nil || len(remaining) != 0 {
		t.Fatalf("expected no orphaned archive tracks, got %v, err %v", remaining, err)
	}
}
