package db_test

import (
	"context"
	"database/sql"
	"errors"
	"server/test/util"
	"testing"
	"time"
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

func TestTouchTrackLastUsed(t *testing.T) {
	ctx := context.Background()
	d := util.OpenTestDB(t)

	track, err := d.CreateTrack(ctx, "youtube", "abc123", "Title", "Artist", "/music/abc123.opus", 180, "https://i.ytimg.com/vi/abc123/hqdefault.jpg")
	if err != nil {
		t.Fatalf("create track: %v", err)
	}
	if track.LastUsedAt == 0 {
		t.Fatalf("expected last_used_at to be set on create, got %v", track.LastUsedAt)
	}

	if err := d.TouchTrackLastUsed(ctx, track.Id); err != nil {
		t.Fatalf("touch last used: %v", err)
	}

	got, err := d.GetTrack(ctx, "abc123")
	if err != nil {
		t.Fatalf("get track: %v", err)
	}
	if got.LastUsedAt < track.LastUsedAt {
		t.Fatalf("expected last_used_at to not decrease after touch, got %v before %v", got.LastUsedAt, track.LastUsedAt)
	}
}

func TestListTracksByLastUsed(t *testing.T) {
	ctx := context.Background()
	d := util.OpenTestDB(t)

	older, err := d.CreateTrack(ctx, "youtube", "older", "Older", "Artist", "/music/older.opus", 180, "")
	if err != nil {
		t.Fatalf("create track: %v", err)
	}
	// unixepoch() has 1-second resolution, so sleep past a tick to guarantee a strictly
	// later last_used_at than "older".
	time.Sleep(1100 * time.Millisecond)
	newer, err := d.CreateTrack(ctx, "youtube", "newer", "Newer", "Artist", "/music/newer.opus", 180, "")
	if err != nil {
		t.Fatalf("create track: %v", err)
	}

	tracks, err := d.ListTracksByLastUsed(ctx)
	if err != nil {
		t.Fatalf("list tracks: %v", err)
	}
	if len(tracks) != 2 || tracks[0].Id != newer.Id || tracks[1].Id != older.Id {
		t.Fatalf("expected [newer, older] order, got %v", tracks)
	}

	// touching "older" should now make it the most recently used.
	time.Sleep(1100 * time.Millisecond)
	if err := d.TouchTrackLastUsed(ctx, older.Id); err != nil {
		t.Fatalf("touch last used: %v", err)
	}

	tracks, err = d.ListTracksByLastUsed(ctx)
	if err != nil {
		t.Fatalf("list tracks: %v", err)
	}
	if len(tracks) != 2 || tracks[0].Id != older.Id || tracks[1].Id != newer.Id {
		t.Fatalf("expected touch to reorder to [older, newer], got %v", tracks)
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
