package session

import (
	"errors"
	db "server/src/db"
	"sync"
)

type SessionQueue struct {
	mu        sync.RWMutex
	tracks    []db.Track
	sessionId string
}

var (
	EmptyQueueError = errors.New("queue is empty")
	BadIndexError   = errors.New("bad queue index")
)

func NewQueue(sessionId string) *SessionQueue {
	return &SessionQueue{
		mu:        sync.RWMutex{},
		tracks:    []db.Track{},
		sessionId: sessionId,
	}
}

func (q *SessionQueue) Enqueue(t db.Track) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.tracks = append(q.tracks, t)
}

func (q *SessionQueue) Dequeue() (db.Track, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.tracks) == 0 {
		return db.Track{}, EmptyQueueError
	}

	t := q.tracks[0]
	q.tracks = q.tracks[1:]

	return t, nil
}

func (q *SessionQueue) ListQueue(sessionId string) ([]db.Track, error) {
	q.mu.RLock()
	defer q.mu.Unlock()

	return q.tracks, nil
}

func (q *SessionQueue) Remove(index uint) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if index > uint(len(q.tracks)) {
		return BadIndexError
	}

	q.tracks = append(q.tracks[:index], q.tracks[index+1:]...)
	return nil
}
