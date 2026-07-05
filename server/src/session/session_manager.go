package session

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

type SessionID string

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[SessionID]*SessionQueue
	Events   chan SessionManagerEvent
}

var (
	AlreadyExistsError       = errors.New("session with provided id already exists")
	SessionNotFoundError = errors.New("session with given ID does not exist")
)

func CreateSessionManager() *SessionManager {
	return &SessionManager{
		mu:       sync.RWMutex{},
		sessions: map[SessionID]*SessionQueue{},
		Events:   make(chan SessionManagerEvent, 16),
	}
}

func (m *SessionManager) CreateSession(ctx context.Context, sessionId SessionID, archiveID *int64) (<-chan error, error) {
	m.mu.Lock()
	_, ok := m.sessions[sessionId]
	if !ok {
		m.sessions[sessionId] = NewQueue(sessionId, archiveID)
		m.mu.Unlock()
	} else {
		m.mu.Unlock()
		return nil, AlreadyExistsError
	}

	slog.Info("session created", "session", sessionId)
	ready := make(chan error, 1)
	select {
	case m.Events <- SessionManagerEvent{Type: SessionCreated, SessionID: sessionId, Ready: ready}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return ready, nil
}

func (m *SessionManager) GetQueue(sessionId SessionID) (*SessionQueue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	queue, ok := m.sessions[sessionId]
	if !ok {
		return nil, SessionNotFoundError
	}
	return queue, nil
}

func (m *SessionManager) ListSessions() []SessionID {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]SessionID, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	return ids
}

func (m *SessionManager) DeleteSession(ctx context.Context, sessionId SessionID) error {
	m.mu.Lock()
	delete(m.sessions, sessionId)
	m.mu.Unlock()

	slog.Info("session deleted", "session", sessionId)
	select {
	case m.Events <- SessionManagerEvent{Type: SessionDeleted, SessionID: sessionId}:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}
