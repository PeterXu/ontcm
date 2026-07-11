package session

import (
	"sync"
	"time"

	"ontcm/internal/knowledge/models"
)

// InMemoryStore implements SessionStore using in-memory map
type InMemoryStore struct {
	sessions map[string]*models.DiagnosticSession
	mutex    sync.RWMutex
	timeout  time.Duration
}

// NewInMemoryStore creates a new in-memory session store
func NewInMemoryStore(timeout time.Duration) *InMemoryStore {
	store := &InMemoryStore{
		sessions: make(map[string]*models.DiagnosticSession),
		timeout:  timeout,
	}

	// Start cleanup goroutine
	go store.cleanupExpiredSessions()

	return store
}

// Create adds a new session to the store
func (s *InMemoryStore) Create(session *models.DiagnosticSession) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.sessions[session.ID] = session
	return nil
}

// Get retrieves a session by ID
func (s *InMemoryStore) Get(id string) (*models.DiagnosticSession, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	session, exists := s.sessions[id]
	if !exists {
		return nil, ErrSessionNotFound
	}

	// Check if expired
	if time.Since(session.UpdatedAt) > s.timeout {
		return nil, ErrSessionExpired
	}

	return session, nil
}

// Update modifies an existing session
func (s *InMemoryStore) Update(id string, session *models.DiagnosticSession) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, exists := s.sessions[id]
	if !exists {
		return ErrSessionNotFound
	}

	session.UpdatedAt = time.Now()
	s.sessions[id] = session

	return nil
}

// Delete removes a session from the store
func (s *InMemoryStore) Delete(id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.sessions, id)
	return nil
}

// Count returns the number of active sessions
func (s *InMemoryStore) Count() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return len(s.sessions)
}

// cleanupExpiredSessions periodically removes expired sessions
func (s *InMemoryStore) cleanupExpiredSessions() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mutex.Lock()
		for id, session := range s.sessions {
			if time.Since(session.UpdatedAt) > s.timeout {
				delete(s.sessions, id)
			}
		}
		s.mutex.Unlock()
	}
}

// Error definitions
var (
	ErrSessionNotFound = &SessionError{Code: "not_found", Message: "Session not found"}
	ErrSessionExpired  = &SessionError{Code: "expired", Message: "Session has expired"}
)

// SessionError represents a session-related error
type SessionError struct {
	Code    string
	Message string
}

func (e *SessionError) Error() string {
	return e.Message
}