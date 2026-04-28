package session

import (
	"context"
	"log/slog"
	"sync"
)

type SessionID string

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[SessionID]*SessionQueue
	Events   chan SessionEvent
}

func CreateSessionManager() *SessionManager {
	return &SessionManager{
		mu:       sync.RWMutex{},
		sessions: map[SessionID]*SessionQueue{},
		Events:   make(chan SessionEvent, 16),
	}
}

func (m *SessionManager) CreateSession(ctx context.Context, sessionId SessionID) error {
	m.mu.Lock()
	m.sessions[sessionId] = NewQueue(sessionId)
	m.mu.Unlock()

	slog.Info("session created", "session", sessionId)
	select {
	case m.Events <- SessionEvent{Type: SessionCreated, SessionID: sessionId}:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
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
	case m.Events <- SessionEvent{Type: SessionDeleted, SessionID: sessionId}:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}
