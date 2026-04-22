package sessions

import (
	"sync"
	"time"

	"github.com/dimmerz92/wares/auth"
)

type SessionStatus int

const (
	StatusUnchanged SessionStatus = iota
	StatusChanged
	StatusDestroy
	StatusShouldRotate
	StatusRotate
)

type session struct {
	ID       string
	Data     map[string]any
	Status   SessionStatus
	Initial  bool
	Lifetime time.Time
	Expiry   time.Time
	mu       sync.RWMutex
}

// Session contains the data and state for the current Session.
type Session struct {
	data *session
}

// NewSession returns a new instance of a Session.
// This session is not set in the context or the session store.
func NewSession(lifetime, expiry time.Time) *Session {
	return &Session{
		data: &session{
			ID:       auth.GenerateURLSafeNonce(32),
			Data:     make(map[string]any),
			Status:   StatusUnchanged,
			Initial:  true,
			Lifetime: lifetime,
			Expiry:   expiry,
		},
	}
}

// ID returns the session ID.
func (s *Session) ID() string {
	s.data.mu.RLock()
	defer s.data.mu.RUnlock()
	return s.data.ID
}

// GetStatus returns the current session status.
func (s *Session) GetStatus() SessionStatus {
	s.data.mu.RLock()
	defer s.data.mu.RUnlock()
	return s.data.Status
}

// SetStatus marks the Session with the given status.
func (s *Session) SetStatus(status SessionStatus) {
	s.data.mu.Lock()
	defer s.data.mu.Unlock()
	s.data.Status = status
}

// Lifetime returns the absolute time at which the session will end.
func (s *Session) Lifetime() time.Time {
	s.data.mu.RLock()
	defer s.data.mu.RUnlock()
	return s.data.Lifetime
}

// Expiry returns the time at which the session will end.
//   - equal to Lifetime if an idle timeout is not set in the SessionManager
//   - not equal to Lifetime if an idle timeout is set in the SessionManager
func (s *Session) Expiry() time.Time {
	s.data.mu.RLock()
	defer s.data.mu.RUnlock()
	return s.data.Expiry
}

// GetValue returns the value mapped to by the given key from the session data if it exists.
func (s *Session) GetValue(key string) any {
	s.data.mu.RLock()
	defer s.data.mu.RUnlock()

	if s.data.Status == StatusDestroy {
		return nil
	}

	return s.data.Data[key]
}

// SetValue adds the key value pair to the session data or overwrites it if it exists.
func (s *Session) SetValue(key string, value any) {
	s.data.mu.Lock()
	defer s.data.mu.Unlock()

	if s.data.Status == StatusDestroy {
		return
	}

	s.data.Data[key] = value

	if s.data.Status != StatusRotate && s.data.Status != StatusShouldRotate {
		s.data.Status = StatusChanged
	}
}

// DeleteValue removes the value mapped to by the given key from the session data if it exists.
func (s *Session) DeleteValue(key string) {
	s.data.mu.Lock()
	defer s.data.mu.Unlock()

	if s.data.Status == StatusDestroy {
		return
	}

	delete(s.data.Data, key)

	if s.data.Status != StatusRotate && s.data.Status != StatusShouldRotate {
		s.data.Status = StatusChanged
	}
}

// Rotate marks the session for rotation at the end of the request cycle.
// All data stored in the current session will be migrated to a new session, with a new ID and expiries.
func (s *Session) Rotate() {
	s.data.mu.Lock()
	defer s.data.mu.Unlock()

	if s.data.Status != StatusDestroy {
		s.data.Status = StatusRotate
	}
}

// Destroy marks the session for destuction at the end of the request cycle.
// Once Destroy is called, the session becomes unusable.
func (s *Session) Destroy() {
	s.data.mu.Lock()
	defer s.data.mu.Unlock()
	s.data.Status = StatusDestroy
}
