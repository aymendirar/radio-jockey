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
	AlreadyExistsError = errors.New("session with provided id already exists")
)

func CreateSessionManager() *SessionManager {
	return &SessionManager{
		mu:       sync.RWMutex{},
		sessions: map[SessionID]*SessionQueue{},
		Events:   make(chan SessionManagerEvent, 16),
	}
}

func (m *SessionManager) CreateSession(ctx context.Context, sessionId SessionID) (SessionID, error) {
	m.mu.Lock()
	_, ok := m.sessions[sessionId]
	if !ok {
		m.sessions[sessionId] = NewQueue(sessionId)
		m.mu.Unlock()
	} else {
		m.mu.Unlock()
		return SessionID(""), AlreadyExistsError
	}

	slog.Info("session created", "session", sessionId)
	select {
	case m.Events <- SessionManagerEvent{Type: SessionCreated, SessionID: sessionId}:
	case <-ctx.Done():
		return SessionID(""), ctx.Err()
	}

	return SessionID(sessionId), nil
}

func (m *SessionManager) GetQueue(sessionId SessionID) *SessionQueue {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[sessionId]
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
