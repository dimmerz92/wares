package memorystore

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type sessionData struct {
	data   []byte
	expiry time.Time
}

// MemoryStore is an in memory backed implementation of sessions.SessionStore.
type MemoryStore struct {
	store    sync.Map
	interval time.Duration
	cancel   context.CancelFunc
}

// SessionStoreOption provides a functional way to configure the MemoryStore.
type SessionStoreOption func(*MemoryStore)

// WithCleanupInterval sets the interval for the store to clear expired sessions. (default: 5 minutes).
//
// To disable the cleanup, set the interval to zero.
func WithCleanupInterval(d time.Duration) SessionStoreOption {
	return func(ms *MemoryStore) {
		if d >= 0 {
			ms.interval = d
		}
	}
}

// NewMemoryStore returns a new instance of the MemoryStore.
//
// Defaults:
//   - cleanup interval: 5 minutes
func NewMemoryStore(opts ...SessionStoreOption) *MemoryStore {
	store := &MemoryStore{interval: 5 * time.Minute}

	for _, opt := range opts {
		opt(store)
	}

	if store.interval > 0 {
		ctx, cancel := context.WithCancel(context.Background())
		store.cancel = cancel
		go store.cleanup(ctx)
	}

	return store
}

// Get retrieves the encoded session data associated with the given key if it exists.
func (ms *MemoryStore) Get(ctx context.Context, key string) (data []byte, ok bool, err error) {
	_ = ctx

	untyped, ok := ms.store.Load(key)
	if !ok {
		return
	}

	session, ok := untyped.(sessionData)
	if !ok {
		err = fmt.Errorf("corrupted session data")
		return
	}

	if time.Now().After(session.expiry) {
		ms.store.Delete(key)
		ok = false
		return
	}

	data = session.data

	return
}

// Set adds the session data to the store or overwrites it if it already exists.
// If the ttl is less than or equal to zero, the data is immediately deleted.
func (ms *MemoryStore) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	_ = ctx

	if ttl <= 0 {
		ms.store.Delete(key)
		return nil
	}

	ms.store.Store(key, sessionData{data: data, expiry: time.Now().Add(ttl)})

	return nil
}

// Delete removes the data associated with the given key if it exists.
func (ms *MemoryStore) Delete(ctx context.Context, key string) error {
	_ = ctx

	ms.store.Delete(key)

	return nil
}

// DeleteMany removes all data associated with the given keys if any exist.
func (ms *MemoryStore) DeleteMany(ctx context.Context, keys ...string) error {
	_ = ctx

	for _, key := range keys {
		ms.store.Delete(key)
	}

	return nil
}

// Stop releases the resources allocated by the MemoryStore.
func (ms *MemoryStore) Stop() {
	if ms.cancel != nil {
		ms.cancel()
	}
}

func (ms *MemoryStore) cleanup(ctx context.Context) {
	ticker := time.NewTicker(ms.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			now := time.Now()
			ms.store.Range(func(key, value any) bool {
				session, ok := value.(sessionData)
				if !ok || now.After(session.expiry) {
					ms.store.Delete(key)
				}
				return true
			})
		}
	}
}
