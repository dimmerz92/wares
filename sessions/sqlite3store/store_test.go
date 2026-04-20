package sqlite3store_test

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/dimmerz92/quicky/sessions/sqlite3store"
	_ "modernc.org/sqlite"
)

func TestSQLite3Store(t *testing.T) {
	db, err := sql.Open("sqlite", "test.db")
	if err != nil {
		t.Fatalf("failed to open memory sqlite db: %v", err)
	}

	store := sqlite3store.NewSQLite3Store(db, sqlite3store.WithCleanupInterval(time.Second))
	defer store.Stop()

	t.Run("Set and Get", func(t *testing.T) {
		tests := []struct {
			name    string
			kvpairs map[string]any
		}{
			{name: "no values", kvpairs: make(map[string]any)},
			{name: "set value", kvpairs: map[string]any{"test": 1}},
			{name: "overwrite value", kvpairs: map[string]any{"test": 42}},
			{name: "set another value", kvpairs: map[string]any{"test2": 99}},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				// Set
				for key, value := range test.kvpairs {
					encoded, _ := json.Marshal(value)
					_ = store.Set(t.Context(), key, encoded, 10*time.Millisecond)
				}

				// Get (exists)
				for key, value := range test.kvpairs {
					encoded, ok, err := store.Get(t.Context(), key)
					if err != nil {
						t.Fatalf("unexpected get error: %v", err)
					}
					if !ok {
						t.Fatal("expected value to exist")
					}

					var got any
					if err := json.Unmarshal(encoded, &got); err != nil {
						t.Fatalf("failed to decode value: %v", err)
					}

					if reflect.DeepEqual(got, value) {
						t.Errorf("expected %v, got %v", value, got)
					}
				}

				// Get (does not exist)
				_, ok, err := store.Get(t.Context(), "doesn't exist")
				if err != nil {
					t.Fatalf("unexpected get error: %v", err)
				}
				if ok {
					t.Error("value should not exist")
				}
			})
		}
	})

	t.Run("Delete", func(t *testing.T) {
		store.Delete(t.Context(), "test")
		_, ok, err := store.Get(t.Context(), "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Error("expected deletion")
		}

		_, ok, err = store.Get(t.Context(), "test2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("expected error to exist")
		}
	})

	t.Run("expired data", func(t *testing.T) {
		time.Sleep(20 * time.Millisecond)
		_, ok, err := store.Get(t.Context(), "test2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Error("value should be expired")
		}
	})

	t.Run("DeleteMany", func(t *testing.T) {
		keys := []string{"test1", "test2", "test3", "test4", "test5"}
		value := []byte("test_value")

		for _, key := range keys {
			_ = store.Set(t.Context(), key, value, time.Minute)
		}

		for _, key := range keys {
			got, ok, err := store.Get(t.Context(), key)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !ok {
				t.Fatal("expected value")
			}

			if !reflect.DeepEqual(got, value) {
				t.Errorf("expected %s, got %s", value, got)
			}
		}

		_ = store.DeleteMany(t.Context(), keys...)

		for _, key := range keys {
			_, ok, err := store.Get(t.Context(), key)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok {
				t.Error("unexpected value")
			}
		}
	})
}
