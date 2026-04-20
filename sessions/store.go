package sessions

import (
	"context"
	"time"
)

// SessionStore defines a storage backend for sessions that maintain user or session data.
type SessionStore interface {
	Get(ctx context.Context, key string) (data []byte, ok bool, err error)
	Set(ctx context.Context, key string, data []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeleteMany(ctx context.Context, keys ...string) error
	Stop()
}
