package music_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"server/src/music"
	"server/test/util"
	"testing"
	"time"
)

func TestCacheEvictsBeyondCacheWindow(t *testing.T) {
	ctx := context.Background()
	d := util.OpenTestDB(t)
	dir := t.TempDir()

	cache, err := music.NewCache(d)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}

	oldCount := 5
	newCount := music.CacheWindow

	oldPaths := make([]string, oldCount)
	for i := range oldCount {
		path := filepath.Join(dir, fmt.Sprintf("old-%d.opus", i))
		if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatalf("write file %d: %v", i, err)
		}
		track, err := d.CreateTrack(ctx, "youtube", fmt.Sprintf("old-%d", i), "Title", "Artist", path, 180, "")
		if err != nil {
			t.Fatalf("create old track %d: %v", i, err)
		}
		if err := cache.Touch(ctx, track, map[int64]struct{}{}); err != nil {
			t.Fatalf("touch old track %d: %v", i, err)
		}
		oldPaths[i] = path
	}

	newPaths := make([]string, newCount)
	for i := range newCount {
		path := filepath.Join(dir, fmt.Sprintf("new-%d.opus", i))
		if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatalf("write file %d: %v", i, err)
		}
		track, err := d.CreateTrack(ctx, "youtube", fmt.Sprintf("new-%d", i), "Title", "Artist", path, 180, "")
		if err != nil {
			t.Fatalf("create new track %d: %v", i, err)
		}
		if err := cache.Touch(ctx, track, map[int64]struct{}{}); err != nil {
			t.Fatalf("touch new track %d: %v", i, err)
		}
		newPaths[i] = path
	}

	for i, path := range oldPaths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected old file %d to be evicted, stat err: %v", i, err)
		}
	}
	for i, path := range newPaths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected new file %d to remain, stat err: %v", i, err)
		}
	}

	for i := range oldCount {
		if _, err := d.GetTrack(ctx, fmt.Sprintf("old-%d", i)); err != nil {
			t.Fatalf("expected evicted track row old-%d to still exist, got err: %v", i, err)
		}
	}
}

func TestCacheSkipsInUseTracks(t *testing.T) {
	ctx := context.Background()
	d := util.OpenTestDB(t)
	dir := t.TempDir()

	cache, err := music.NewCache(d)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}

	oldPath := filepath.Join(dir, "old.opus")
	if err := os.WriteFile(oldPath, []byte("data"), 0644); err != nil {
		t.Fatalf("write old file: %v", err)
	}
	oldTrack, err := d.CreateTrack(ctx, "youtube", "old", "Title", "Artist", oldPath, 180, "")
	if err != nil {
		t.Fatalf("create old track: %v", err)
	}
	inUse := map[int64]struct{}{oldTrack.Id: {}}
	if err := cache.Touch(ctx, oldTrack, inUse); err != nil {
		t.Fatalf("touch old track: %v", err)
	}

	for i := range music.CacheWindow {
		path := filepath.Join(dir, fmt.Sprintf("new-%d.opus", i))
		if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatalf("write file %d: %v", i, err)
		}
		track, err := d.CreateTrack(ctx, "youtube", fmt.Sprintf("new-%d", i), "Title", "Artist", path, 180, "")
		if err != nil {
			t.Fatalf("create new track %d: %v", i, err)
		}
		if err := cache.Touch(ctx, track, inUse); err != nil {
			t.Fatalf("touch new track %d: %v", i, err)
		}
	}

	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("expected in-use track's file to be preserved, stat err: %v", err)
	}
}

func TestCacheNoopUnderWindow(t *testing.T) {
	ctx := context.Background()
	d := util.OpenTestDB(t)
	dir := t.TempDir()

	cache, err := music.NewCache(d)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}

	path := filepath.Join(dir, "track.opus")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	track, err := d.CreateTrack(ctx, "youtube", "id-0", "Title", "Artist", path, 180, "")
	if err != nil {
		t.Fatalf("create track: %v", err)
	}
	if err := cache.Touch(ctx, track, map[int64]struct{}{}); err != nil {
		t.Fatalf("touch track: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to remain when under the cache window, stat err: %v", err)
	}
}

// TestNewCacheHydratesFromExistingRows simulates a server restart: CacheWindow tracks
// already exist in the database from "before the restart" (so a well-behaved prior process
// would never have needed to evict any of them), and a fresh Cache built over that same
// database must reconstruct their recency order purely from last_used_at, then continue
// enforcing the window correctly as more tracks are touched after the restart.
func TestNewCacheHydratesFromExistingRows(t *testing.T) {
	ctx := context.Background()
	d := util.OpenTestDB(t)
	dir := t.TempDir()

	staleCount := 10
	freshCount := music.CacheWindow - staleCount
	postRestartCount := staleCount

	stalePaths := make([]string, staleCount)
	for i := range staleCount {
		path := filepath.Join(dir, fmt.Sprintf("stale-%d.opus", i))
		if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatalf("write file %d: %v", i, err)
		}
		if _, err := d.CreateTrack(ctx, "youtube", fmt.Sprintf("stale-%d", i), "Title", "Artist", path, 180, ""); err != nil {
			t.Fatalf("create stale track %d: %v", i, err)
		}
		stalePaths[i] = path
	}

	// unixepoch() has 1-second resolution: a single sleep here guarantees every "fresh" track
	// below sorts strictly after every "stale" track above, without needing a sleep per track.
	time.Sleep(1100 * time.Millisecond)

	freshPaths := make([]string, freshCount)
	for i := range freshCount {
		path := filepath.Join(dir, fmt.Sprintf("fresh-%d.opus", i))
		if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatalf("write file %d: %v", i, err)
		}
		if _, err := d.CreateTrack(ctx, "youtube", fmt.Sprintf("fresh-%d", i), "Title", "Artist", path, 180, ""); err != nil {
			t.Fatalf("create fresh track %d: %v", i, err)
		}
		freshPaths[i] = path
	}
	// staleCount + freshCount == CacheWindow, so a well-behaved prior process would never
	// have evicted any of the above, and both groups' files still exist on disk here.

	// a fresh Cache, built without any Touch calls, must reconstruct its LRU state purely
	// from last_used_at already present in the database (as if this were a server restart).
	cache, err := music.NewCache(d)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}

	postRestartPaths := make([]string, postRestartCount)
	for i := range postRestartCount {
		path := filepath.Join(dir, fmt.Sprintf("post-%d.opus", i))
		if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatalf("write file %d: %v", i, err)
		}
		track, err := d.CreateTrack(ctx, "youtube", fmt.Sprintf("post-%d", i), "Title", "Artist", path, 180, "")
		if err != nil {
			t.Fatalf("create post-restart track %d: %v", i, err)
		}
		if err := cache.Touch(ctx, track, map[int64]struct{}{}); err != nil {
			t.Fatalf("touch post-restart track %d: %v", i, err)
		}
		postRestartPaths[i] = path
	}

	for i, path := range stalePaths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected stale file %d to be evicted, stat err: %v", i, err)
		}
	}
	for i, path := range freshPaths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected fresh file %d (hydrated from the database) to survive, stat err: %v", i, err)
		}
	}
	for i, path := range postRestartPaths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected post-restart file %d to survive, stat err: %v", i, err)
		}
	}
}

func TestCacheGetHit(t *testing.T) {
	ctx := context.Background()
	d := util.OpenTestDB(t)
	dir := t.TempDir()

	cache, err := music.NewCache(d)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}

	path := filepath.Join(dir, "track.opus")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	track, err := d.CreateTrack(ctx, "youtube", "abc123", "Title", "Artist", path, 180, "")
	if err != nil {
		t.Fatalf("create track: %v", err)
	}
	if err := cache.Touch(ctx, track, map[int64]struct{}{}); err != nil {
		t.Fatalf("touch track: %v", err)
	}

	got, ok := cache.Get("abc123")
	if !ok {
		t.Fatal("expected a hit for a touched track")
	}
	if got.Id != track.Id {
		t.Fatalf("expected track %d, got %d", track.Id, got.Id)
	}
}

func TestCacheGetMiss(t *testing.T) {
	d := util.OpenTestDB(t)

	cache, err := music.NewCache(d)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}

	if _, ok := cache.Get("nonexistent"); ok {
		t.Fatal("expected a miss for a source id that was never touched")
	}
}

func TestCacheGetAfterEviction(t *testing.T) {
	ctx := context.Background()
	d := util.OpenTestDB(t)
	dir := t.TempDir()

	cache, err := music.NewCache(d)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}

	oldPath := filepath.Join(dir, "old.opus")
	if err := os.WriteFile(oldPath, []byte("data"), 0644); err != nil {
		t.Fatalf("write old file: %v", err)
	}
	oldTrack, err := d.CreateTrack(ctx, "youtube", "old", "Title", "Artist", oldPath, 180, "")
	if err != nil {
		t.Fatalf("create old track: %v", err)
	}
	if err := cache.Touch(ctx, oldTrack, map[int64]struct{}{}); err != nil {
		t.Fatalf("touch old track: %v", err)
	}

	for i := range music.CacheWindow {
		path := filepath.Join(dir, fmt.Sprintf("new-%d.opus", i))
		if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatalf("write file %d: %v", i, err)
		}
		track, err := d.CreateTrack(ctx, "youtube", fmt.Sprintf("new-%d", i), "Title", "Artist", path, 180, "")
		if err != nil {
			t.Fatalf("create new track %d: %v", i, err)
		}
		if err := cache.Touch(ctx, track, map[int64]struct{}{}); err != nil {
			t.Fatalf("touch new track %d: %v", i, err)
		}
	}

	if _, ok := cache.Get("old"); ok {
		t.Fatal("expected a miss for a source id that was evicted")
	}
}

func TestCacheGetAfterHydration(t *testing.T) {
	ctx := context.Background()
	d := util.OpenTestDB(t)
	dir := t.TempDir()

	path := filepath.Join(dir, "track.opus")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	track, err := d.CreateTrack(ctx, "youtube", "abc123", "Title", "Artist", path, 180, "")
	if err != nil {
		t.Fatalf("create track: %v", err)
	}

	// no Touch call here — the track's presence in the database alone (as if from a prior
	// process, before this restart) must be enough for Get to find it after hydration.
	cache, err := music.NewCache(d)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}

	got, ok := cache.Get("abc123")
	if !ok {
		t.Fatal("expected Get to find a track hydrated from the database")
	}
	if got.Id != track.Id {
		t.Fatalf("expected track %d, got %d", track.Id, got.Id)
	}
}

func TestCacheGetProtectsFromEviction(t *testing.T) {
	ctx := context.Background()
	d := util.OpenTestDB(t)
	dir := t.TempDir()

	cache, err := music.NewCache(d)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}

	aPath := filepath.Join(dir, "a.opus")
	if err := os.WriteFile(aPath, []byte("data"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	trackA, err := d.CreateTrack(ctx, "youtube", "a", "Title", "Artist", aPath, 180, "")
	if err != nil {
		t.Fatalf("create track a: %v", err)
	}
	if err := cache.Touch(ctx, trackA, map[int64]struct{}{}); err != nil {
		t.Fatalf("touch track a: %v", err)
	}

	// fill the rest of the window so A is the LRU tail.
	for i := range music.CacheWindow - 1 {
		path := filepath.Join(dir, fmt.Sprintf("filler-%d.opus", i))
		if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatalf("write file %d: %v", i, err)
		}
		track, err := d.CreateTrack(ctx, "youtube", fmt.Sprintf("filler-%d", i), "Title", "Artist", path, 180, "")
		if err != nil {
			t.Fatalf("create filler track %d: %v", i, err)
		}
		if err := cache.Touch(ctx, track, map[int64]struct{}{}); err != nil {
			t.Fatalf("touch filler track %d: %v", i, err)
		}
	}

	// Get moves A back to the front, protecting it from the very next eviction.
	if _, ok := cache.Get("a"); !ok {
		t.Fatal("expected a hit for track a")
	}

	newPath := filepath.Join(dir, "new.opus")
	if err := os.WriteFile(newPath, []byte("data"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	newTrack, err := d.CreateTrack(ctx, "youtube", "new", "Title", "Artist", newPath, 180, "")
	if err != nil {
		t.Fatalf("create new track: %v", err)
	}
	if err := cache.Touch(ctx, newTrack, map[int64]struct{}{}); err != nil {
		t.Fatalf("touch new track: %v", err)
	}

	if _, err := os.Stat(aPath); err != nil {
		t.Fatalf("expected track a's file to survive after Get moved it to the front, stat err: %v", err)
	}
}
