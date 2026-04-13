package session

import "sync"

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*SessionQueue
	Events   chan SessionEvent
}

func CreateSessionManager() *SessionManager {
	return &SessionManager{
		mu:       sync.RWMutex{},
		sessions: map[string]*SessionQueue{},
		Events:   make(chan SessionEvent, 16),
	}
}

func (m *SessionManager) CreateSession(sessionId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events <- SessionEvent{Type: SessionCreated, SessionID: sessionId}
	return nil
}

func (m *SessionManager) DeleteSession(sessionId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events <- SessionEvent{Type: SessionDeleted, SessionID: sessionId}
	return nil
}

func (m *SessionManager) ClearEmptySessions() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return nil
}
