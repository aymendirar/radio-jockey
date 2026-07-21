package music

import (
	"container/list"
	"context"
	"log/slog"
	"os"
	"sync"

	"server/src/db"
)

// yt-dlp skips downloading when the destination file already exists, and
// re-downloads it when missing, so deleting a track's file here is safe.
const CacheWindow = 50

type Cache struct {
	mu          sync.Mutex
	db          *db.DB
	window      *list.List
	windowNodes map[int64]*list.Element
	bySourceID  map[string]*list.Element
}

func NewCache(d *db.DB) (*Cache, error) {
	tracks, err := d.ListTracksByLastUsed(context.Background())
	if err != nil {
		return nil, err
	}
	if len(tracks) > CacheWindow {
		tracks = tracks[:CacheWindow]
	}

	c := &Cache{
		db:          d,
		window:      list.New(),
		windowNodes: make(map[int64]*list.Element),
		bySourceID:  make(map[string]*list.Element),
	}
	for _, t := range tracks {
		el := c.window.PushBack(t)
		c.windowNodes[t.ID] = el
		c.bySourceID[t.SourceID] = el
	}
	return c, nil
}

// Get looks up a track by its source id without mutating recency state on disk or in the
// database; a hit moves the track to the front of the window so a concurrent Touch/Evict
// can't evict it out from under the caller before it's actually enqueued and re-touched.
func (c *Cache) Get(sourceID string) (*db.Track, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.bySourceID[sourceID]
	if !ok {
		return nil, false
	}
	c.window.MoveToFront(el)
	return el.Value.(*db.Track), true
}

// Touch marks a track as recently used and evicts the oldest unused tracks when the
// cache exceeds CacheWindow. Evicted tracks have their files deleted from disk; tracks
// currently in a session queue are never evicted. The caller must pass the set of
// in-use track IDs across all active sessions.
func (c *Cache) Touch(ctx context.Context, track *db.Track, inUse map[int64]struct{}) error {
	if err := c.db.TouchTrackLastUsed(ctx, track.ID); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.windowNodes[track.ID]; ok {
		el.Value = track
		c.window.MoveToFront(el)
	} else {
		el := c.window.PushFront(track)
		c.windowNodes[track.ID] = el
		c.bySourceID[track.SourceID] = el
	}

	for c.window.Len() > CacheWindow {
		victim := c.window.Back()
		// find oldest track that is not in use (currently playing or in queue)
		for victim != nil {
			if _, used := inUse[victim.Value.(*db.Track).ID]; !used {
				break
			}
			victim = victim.Prev()
		}
		if victim == nil {
			break
		}

		evicted := victim.Value.(*db.Track)
		c.window.Remove(victim)
		delete(c.windowNodes, evicted.ID)
		delete(c.bySourceID, evicted.SourceID)
		if err := os.Remove(evicted.FilePath); err != nil && !os.IsNotExist(err) {
			slog.Warn("failed to evict cached track file", "err", err, "path", evicted.FilePath)
		}
	}
	return nil
}
