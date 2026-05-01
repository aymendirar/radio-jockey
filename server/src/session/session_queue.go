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
	Events    chan SessionQueueEvent
}

const MaxQueueSize = 16

var (
	EmptyQueueError = errors.New("queue is empty")
	BadIndexError   = errors.New("bad queue index")
	FullQueueError  = errors.New("queue is full")
)

func NewQueue(sessionId SessionID) *SessionQueue {
	return &SessionQueue{
		mu:        sync.RWMutex{},
		tracks:    make([]*db.Track, 0, MaxQueueSize),
		sessionId: sessionId,
		notify:    make(chan struct{}, 1),
		Events:    make(chan SessionQueueEvent, 1),
	}
}

func (q *SessionQueue) Skip() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.tracks) == 0 {
		return EmptyQueueError
	}

	select {
	case q.Events <- SessionQueueEvent{Type: SkipTrack}:
	default:
	}
	return nil
}

func (q *SessionQueue) Enqueue(t *db.Track) error {
	q.mu.Lock()
	if len(q.tracks) >= MaxQueueSize {
		q.mu.Unlock()
		return FullQueueError
	}
	q.tracks = append(q.tracks, t)
	q.mu.Unlock()

	select {
	case q.notify <- struct{}{}:
	default:
	}
	return nil
}

func (q *SessionQueue) Peek() (*db.Track, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if len(q.tracks) == 0 {
		return nil, EmptyQueueError
	}

	return q.tracks[0], nil
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
