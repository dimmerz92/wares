package sqlite3store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

type Statement int

const (
	Get Statement = iota
	Set
	Delete
	DeleteMany
	DeleteExpired
)

var validTablePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// SQLite3Store is an SQLite3 backed implementation of sessions.SessionStore.
type SQLite3Store struct {
	db       *sql.DB
	table    string
	stmts    map[Statement]*sql.Stmt
	interval time.Duration
	cancel   context.CancelFunc
}

// SessionStoreOption provides a functional way to configure the SQLite3Store.
type SessionStoreOption func(*SQLite3Store)

// WithCleanupInterval sets the interval for the store to clear expired sessions. (default: 5 minutes).
//
// To disable the cleanup, set the interval to zero.
func WithCleanupInterval(d time.Duration) SessionStoreOption {
	return func(ms *SQLite3Store) {
		if d >= 0 {
			ms.interval = d
		}
	}
}

// WithTableName sets the name of the session table to be migrated to the database. (default: __quicky_sessions).
//
// Panics if the given name is invalid.
func WithTableName(name string) SessionStoreOption {
	return func(ss *SQLite3Store) {
		if strings.HasPrefix(name, "sqlite_") || !validTablePattern.MatchString(name) {
			panic(fmt.Sprintf("invalid sqlite3 table name: %s", name))
		}
		ss.table = name
	}
}

// NewSQLite3Store returns a new instance of the SQLite3Store.
//
// Panics if the db is nil.
//
// Defaults:
//   - cleanup interval: 5 minutes
//   - table name: __quicky_sessions
func NewSQLite3Store(db *sql.DB, opts ...SessionStoreOption) *SQLite3Store {
	store := &SQLite3Store{
		db:       db,
		interval: 5 * time.Minute,
		table:    "__quicky_sessions",
		stmts:    make(map[Statement]*sql.Stmt),
	}

	for _, opt := range opts {
		opt(store)
	}

	_, err := db.Exec(
		fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id TEXT PRIMARY KEY, data BLOB, expiry TIMESTAMP)", store.table),
	)
	if err != nil {
		panic(err)
	}

	stmts := map[Statement]string{
		Get:           "SELECT * FROM %s WHERE id = ?",
		Set:           "INSERT INTO %s (id, data, expiry) VALUES (?1, ?2, ?3) ON CONFLICT DO UPDATE SET data = ?2, expiry = ?3 WHERE id = ?1",
		Delete:        "DELETE FROM %s WHERE id = ?",
		DeleteMany:    "DELETE FROM %s WHERE id IN (SELECT value FROM json_each(?))",
		DeleteExpired: "DELETE FROM %s WHERE expiry < ?",
	}

	for mapping, query := range stmts {
		stmt, err := db.Prepare(fmt.Sprintf(query, store.table))
		if err != nil {
			panic(err)
		}
		store.stmts[mapping] = stmt
	}

	if store.interval > 0 {
		ctx, cancel := context.WithCancel(context.Background())
		store.cancel = cancel
		go store.cleanup(ctx)
	}

	return store
}

// Get retrieves the encoded session data associated with the given key if it exists.
func (ss *SQLite3Store) Get(ctx context.Context, key string) (data []byte, ok bool, err error) {
	var (
		id     string
		expiry time.Time
	)

	err = ss.stmts[Get].QueryRowContext(ctx, key).Scan(&id, &data, &expiry)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = nil
		}
		return
	}

	if time.Now().After(expiry) {
		ss.Delete(ctx, key)
		ok = false
		return
	}

	ok = true

	return
}

// Set adds the session data to the store or overwrites it if it already exists.
// If the ttl is less than or equal to zero, the data is immediately deleted.
func (ss *SQLite3Store) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	if ttl <= 0 {
		ss.Delete(ctx, key)
		return nil
	}

	_, err := ss.stmts[Set].ExecContext(ctx, key, data, time.Now().Add(ttl))

	return err
}

// Delete removes the data associated with the given key if it exists.
func (ss *SQLite3Store) Delete(ctx context.Context, key string) error {
	_, err := ss.stmts[Delete].ExecContext(ctx, key)

	return err
}

// DeleteMany removes all data associated with the given keys if any exist.
func (ss *SQLite3Store) DeleteMany(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	encoded, err := json.Marshal(keys)
	if err != nil {
		return err
	}

	_, err = ss.stmts[DeleteMany].ExecContext(ctx, string(encoded))

	return err
}

// Stop releases the resources allocated by the SQLite3Store
func (ss *SQLite3Store) Stop() {
	if ss.cancel != nil {
		ss.cancel()
	}
}

func (ss *SQLite3Store) cleanup(ctx context.Context) {
	ticker := time.NewTicker(ss.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			_, err := ss.stmts[DeleteExpired].ExecContext(ctx, time.Now())
			if err != nil {
				slog.Error("sqlite3store", slog.Any("cleanup error", err))
			}
		}
	}
}
