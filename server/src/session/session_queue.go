package session

import (
	"errors"
	db "server/src/db"
	"sync"
)

type SessionQueue struct {
	mu        sync.RWMutex
	tracks    []*db.Track
	sessionId SessionID
	notify    chan struct{}
}

var (
	EmptyQueueError = errors.New("queue is empty")
	BadIndexError   = errors.New("bad queue index")
)

func NewQueue(sessionId SessionID) *SessionQueue {
	return &SessionQueue{
		mu:        sync.RWMutex{},
		tracks:    []*db.Track{},
		sessionId: sessionId,
		notify:    make(chan struct{}, 1),
	}
}

func (q *SessionQueue) Enqueue(t *db.Track) {
	q.mu.Lock()
	q.tracks = append(q.tracks, t)
	q.mu.Unlock()

	select {
	case q.notify <- struct{}{}:
	default:
	}
}

func (q *SessionQueue) Dequeue() (*db.Track, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.tracks) == 0 {
		return nil, EmptyQueueError
	}

	t := q.tracks[0]
	q.tracks = q.tracks[1:]
	return t, nil
}

func (q *SessionQueue) Notify() <-chan struct{} {
	return q.notify
}

func (q *SessionQueue) ListQueue() ([]*db.Track, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.tracks, nil
}

func (q *SessionQueue) Remove(index uint) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if index >= uint(len(q.tracks)) {
		return BadIndexError
	}

	q.tracks = append(q.tracks[:index], q.tracks[index+1:]...)
	return nil
}
