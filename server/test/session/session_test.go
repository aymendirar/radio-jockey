package session_test

import (
	"context"
	"server/src/db"
	"server/src/session"
	"testing"
)

func newTestQueue() *session.SessionQueue {
	archiveID := int64(42)
	return session.NewQueue("test-session", &archiveID)
}

func TestSessionQueueArchiveID(t *testing.T) {
	q := newTestQueue()
	if *q.ArchiveID() != 42 {
		t.Fatalf("expected archiveID 42, got %v", q.ArchiveID())
	}

	unarchived := session.NewQueue("test-session", nil)
	if unarchived.ArchiveID() != nil {
		t.Fatalf("expected nil archiveID, got %v", unarchived.ArchiveID())
	}
}

func TestSessionQueueEmpty(t *testing.T) {
	q := newTestQueue()

	if _, err := q.Peek(); err != session.EmptyQueueError {
		t.Fatalf("expected EmptyQueueError on empty peek, got %v", err)
	}
	if err := q.Skip(); err != session.EmptyQueueError {
		t.Fatalf("expected EmptyQueueError on empty skip, got %v", err)
	}
}

func TestSessionQueueEnqueuePeek(t *testing.T) {
	q := newTestQueue()

	track1 := &db.Track{Id: 1, Title: "one"}
	track2 := &db.Track{Id: 2, Title: "two"}
	if err := q.Enqueue(track1); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := q.Enqueue(track2); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if peeked, err := q.Peek(); err != nil || peeked.Id != track1.Id {
		t.Fatalf("expected to peek track1, got %v, err %v", peeked, err)
	}
}

func TestSessionQueueRemove(t *testing.T) {
	q := newTestQueue()
	q.Enqueue(&db.Track{Id: 1})
	q.Enqueue(&db.Track{Id: 2})

	if err := q.Remove(5); err != session.BadIndexError {
		t.Fatalf("expected BadIndexError, got %v", err)
	}
	if err := q.Remove(1); err != nil {
		t.Fatalf("remove: %v", err)
	}

	tracks, _ := q.ListQueue()
	if len(tracks) != 1 || tracks[0].Id != 1 {
		t.Fatalf("expected only track 1 left, got %v", tracks)
	}
}

func TestSessionQueueSkip(t *testing.T) {
	q := newTestQueue()
	q.Enqueue(&db.Track{Id: 1})

	if err := q.Skip(); err != nil {
		t.Fatalf("skip: %v", err)
	}
	select {
	case event := <-q.Events:
		if event.Type != session.SkipTrack {
			t.Fatalf("expected SkipTrack event, got %v", event.Type)
		}
	default:
		t.Fatal("expected a SkipTrack event to be queued")
	}
}

func TestSessionQueueDequeue(t *testing.T) {
	q := newTestQueue()
	q.Enqueue(&db.Track{Id: 1})

	dequeued, err := q.Dequeue()
	if err != nil || dequeued.Id != 1 {
		t.Fatalf("expected to dequeue track1, got %v, err %v", dequeued, err)
	}
	if _, err := q.Dequeue(); err != session.EmptyQueueError {
		t.Fatalf("expected EmptyQueueError after draining queue, got %v", err)
	}
}

func TestSessionQueueFull(t *testing.T) {
	q := newTestQueue()

	for i := range session.MaxQueueSize {
		if err := q.Enqueue(&db.Track{Id: int64(i)}); err != nil {
			t.Fatalf("enqueue %d: %v", i, err)
		}
	}
	if err := q.Enqueue(&db.Track{Id: 999}); err != session.FullQueueError {
		t.Fatalf("expected FullQueueError once queue is full, got %v", err)
	}
}

func TestSessionManagerCreateDuplicate(t *testing.T) {
	ctx := context.Background()
	m := session.CreateSessionManager(1000)

	if _, err := m.CreateSession(ctx, "s1", nil); err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := m.CreateSession(ctx, "s1", nil); err != session.AlreadyExistsError {
		t.Fatalf("expected AlreadyExistsError, got %v", err)
	}
}

func TestSessionManagerTooManySessions(t *testing.T) {
	ctx := context.Background()
	m := session.CreateSessionManager(2)

	if _, err := m.CreateSession(ctx, "s1", nil); err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := m.CreateSession(ctx, "s2", nil); err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := m.CreateSession(ctx, "s3", nil); err != session.TooManySessionsError {
		t.Fatalf("expected TooManySessionsError, got %v", err)
	}
}

func TestSessionManagerGetQueue(t *testing.T) {
	ctx := context.Background()
	m := session.CreateSessionManager(1000)

	archiveID := int64(7)
	if _, err := m.CreateSession(ctx, "s1", &archiveID); err != nil {
		t.Fatalf("create session: %v", err)
	}

	queue, err := m.GetQueue("s1")
	if err != nil {
		t.Fatalf("get queue: %v", err)
	}
	if *queue.ArchiveID() != archiveID {
		t.Fatalf("expected archiveID %d on queue, got %v", archiveID, queue.ArchiveID())
	}

	if _, err := m.GetQueue("nonexistent"); err != session.SessionNotFoundError {
		t.Fatalf("expected SessionNotFoundError, got %v", err)
	}
}

func TestSessionManagerListSessions(t *testing.T) {
	ctx := context.Background()
	m := session.CreateSessionManager(1000)

	if len(m.ListSessions()) != 0 {
		t.Fatalf("expected no sessions initially, got %v", m.ListSessions())
	}

	m.CreateSession(ctx, "s1", nil)
	m.CreateSession(ctx, "s2", nil)

	ids := m.ListSessions()
	if len(ids) != 2 {
		t.Fatalf("expected 2 sessions, got %v", ids)
	}
}

func TestSessionManagerInUseTrackIDs(t *testing.T) {
	ctx := context.Background()
	m := session.CreateSessionManager(1000)

	if ids := m.InUseTrackIDs(); len(ids) != 0 {
		t.Fatalf("expected no in-use tracks initially, got %v", ids)
	}

	m.CreateSession(ctx, "s1", nil)
	m.CreateSession(ctx, "s2", nil)

	q1, err := m.GetQueue("s1")
	if err != nil {
		t.Fatalf("get queue: %v", err)
	}
	q1.Enqueue(&db.Track{Id: 1})
	q1.Enqueue(&db.Track{Id: 2})

	q2, err := m.GetQueue("s2")
	if err != nil {
		t.Fatalf("get queue: %v", err)
	}
	q2.Enqueue(&db.Track{Id: 3})

	ids := m.InUseTrackIDs()
	if len(ids) != 3 {
		t.Fatalf("expected 3 in-use tracks, got %v", ids)
	}
	for _, id := range []int64{1, 2, 3} {
		if _, ok := ids[id]; !ok {
			t.Fatalf("expected track %d to be in use, got %v", id, ids)
		}
	}
}

func TestSessionManagerDeleteSession(t *testing.T) {
	ctx := context.Background()
	m := session.CreateSessionManager(1000)
	m.CreateSession(ctx, "s1", nil)

	if err := m.DeleteSession(ctx, "s1"); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if _, err := m.GetQueue("s1"); err != session.SessionNotFoundError {
		t.Fatalf("expected SessionNotFoundError after delete, got %v", err)
	}
	if len(m.ListSessions()) != 0 {
		t.Fatalf("expected no sessions after delete, got %v", m.ListSessions())
	}
}
