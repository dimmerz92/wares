package sessions

import (
	"context"
	"errors"
	"maps"
	"time"

	"github.com/dimmerz92/wares/auth"
)

var ErrNilSession = errors.New("nil session")

// SessionManager manages session lifecycles using a SessionStore backend.
type SessionManager struct {
	ctxKey          string
	encoder         Encoder
	store           SessionStore
	lifetime        time.Duration
	idleTimeout     time.Duration
	rotateThreshold time.Duration
}

// SessionManagerOption provides a functional way to configure the SessionManager on construction.
type SessionManagerOption func(*SessionManager)

// WithContextKey sets the context key to be used when setting the session into the request context.
// Defaults to a randomly generated string. Blank strings will be ignored and the default used.
func WithContextKey(key string) SessionManagerOption {
	return func(sm *SessionManager) {
		if key != "" {
			sm.ctxKey = key
		}
	}
}

// WithEncoder sets the encoder used to serialise data prior to storage.
// Defaults to a GobEncoder. Nil values will be ignored and the default used.
func WithEncoder(encoder Encoder) SessionManagerOption {
	return func(sm *SessionManager) { sm.encoder = encoder }
}

// WithStore sets the SessionStore to be used to persist session data.
// Defaults to a memorystore.MemoryStore. Nil values will be ignored and the defaul used.
func WithStore(store SessionStore) SessionManagerOption {
	return func(sm *SessionManager) { sm.store = store }
}

// WithLifetime sets the absolute expiry for a session to be force expired at if not rotated.
// Defaults to 24 hours. Values less than or equal to zero are ignored.
func WithLifetime(d time.Duration) SessionManagerOption {
	return func(sm *SessionManager) {
		if d > 0 {
			sm.lifetime = d
		}
	}
}

// WithIdleTimeout sets an optional idle timeout for a session to be force expired at if not used within the duration.
// No default set. Values less than or equal to zero will be ignored.
func WithIdleTimeout(d time.Duration) SessionManagerOption {
	return func(sm *SessionManager) {
		if d > 0 {
			sm.idleTimeout = d
		}
	}
}

// WithRotateThreshold sets the threshold the SessionManager uses to set the StatusShouldRotate flag on the session.
// Defaults to 2 minutes remaining on the existing session. Values less than or equal to zero will be ignored.
func WithRotateThreshold(d time.Duration) SessionManagerOption {
	return func(sm *SessionManager) {
		if d > 0 {
			sm.rotateThreshold = d
		}
	}
}

// NewSessionManager creates a new instance of a SessionManager with the given config options or defaults.
//
// Defaults:
//   - context key: random string
//   - encoder: gob based encoding
//   - store: none
//   - lifetime: 24 hours
//   - idle timeout: none
//   - rotate threshold: 2 minutes
func NewSessionManager(opts ...SessionManagerOption) *SessionManager {
	manager := &SessionManager{
		ctxKey:          auth.GenerateURLSafeNonce(16),
		lifetime:        24 * time.Hour,
		rotateThreshold: 2 * time.Minute,
	}

	for _, opt := range opts {
		opt(manager)
	}

	if manager.encoder == nil {
		manager.encoder = NewGobEncoder()
	}

	return manager
}

// NewSession returns a new unique instance of a Session.
// The returned Session is not added to the SessionStore and must be commited before the request lifecycle completes.
func (sm *SessionManager) NewSession() *Session {
	now := time.Now()
	lifetime := now.Add(sm.lifetime)
	expiry := lifetime

	if sm.idleTimeout > 0 && time.Until(lifetime) > sm.idleTimeout {
		expiry = now.Add(sm.idleTimeout)
	}

	session := NewSession(lifetime, expiry)

	return session
}

// GetSession retrieves a session by id if it exists.
func (sm *SessionManager) GetSession(ctx context.Context, sessionId string) (session *Session, ok bool, err error) {
	encoded, ok, err := sm.store.Get(ctx, sessionId)
	if err != nil || !ok {
		return
	}

	session = &Session{}
	err = sm.encoder.Unmarshal(encoded, &session.data)
	if err != nil {
		ok = false
		return
	}

	now := time.Now()
	if now.After(session.data.Expiry) || now.After(session.data.Lifetime) {
		ok = false
		return
	}

	if time.Until(session.data.Lifetime) <= sm.rotateThreshold {
		session.data.Status = StatusShouldRotate
	}

	return
}

// SetSession updates the Session expiry automatically and adds it to the store, or updates if it already exists.
func (sm *SessionManager) SetSession(ctx context.Context, session *Session) error {
	if session == nil {
		return ErrNilSession
	}

	ttl := time.Until(session.data.Lifetime)
	if sm.idleTimeout > 0 {
		ttl = sm.idleTimeout
		session.data.Expiry = time.Now().Add(ttl)
	}

	encoded, err := sm.encoder.Marshal(session.data)
	if err != nil {
		return err
	}

	return sm.store.Set(ctx, session.data.ID, encoded, ttl)
}

// DeleteSession removes the session associated with the given session ID from the store if it exists.
func (sm *SessionManager) DeleteSession(ctx context.Context, sessionId string) error {
	return sm.store.Delete(ctx, sessionId)
}

// DeleteManySessions removes all sessions associated with the given session IDs from the store if any exist.
func (sm *SessionManager) DeleteManySessions(ctx context.Context, sessionIds ...string) error {
	return sm.store.DeleteMany(ctx, sessionIds...)
}

// Commit pushes all changes made to the session into the SessionStore.
// If no changes were made and an idle timeout is configured, Commit will increment the expiry to the correct time.
func (sm *SessionManager) Commit(ctx context.Context, session *Session) error {
	if session == nil {
		return ErrNilSession
	}

	switch session.data.Status {
	case StatusDestroy:
		err := sm.DeleteSession(ctx, session.data.ID)
		if err != nil {
			return err
		}

	case StatusRotate:
		err := sm.DeleteSession(ctx, session.data.ID)
		if err != nil {
			return err
		}

		newSession := sm.NewSession()
		newSession.data.Data = maps.Clone(session.data.Data)
		newSession.data.Status = StatusChanged
		session = newSession

		fallthrough

	default:
		if session.data.Status == StatusUnchanged && sm.idleTimeout == 0 {
			return nil
		}

		err := sm.SetSession(ctx, session)
		if err != nil {
			return err
		}
	}

	return nil
}
