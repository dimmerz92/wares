package sessions_test

import (
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/dimmerz92/wares/sessions"
	"github.com/dimmerz92/wares/sessions/memorystore"
)

func TestSessionManager(t *testing.T) {
	store := memorystore.NewMemoryStore()
	manager := sessions.NewSessionManager(
		sessions.WithStore(store),
		sessions.WithRotateThreshold(20*time.Millisecond),
	)
	defer store.Stop()

	sess := make(map[string]*sessions.Session)

	t.Run("NewSession", func(t *testing.T) {
		expiry := time.Now().Add(100 * time.Millisecond)
		for range 2 {
			session := sessions.NewSession(expiry, expiry)
			sess[session.ID()] = session
		}
	})

	t.Run("SetSession", func(t *testing.T) {
		for _, session := range sess {
			err := manager.SetSession(t.Context(), session)
			if err != nil {
				t.Fatalf("unexpected error: %#v", err)
			}
		}
	})

	t.Run("GetSession", func(t *testing.T) {
		for key, session := range sess {
			got, ok, err := manager.GetSession(t.Context(), key)
			if err != nil {
				t.Fatalf("unexpected error: %#v", err)
			}

			if !ok {
				t.Error("expected session")
			}

			if got.ID() != session.ID() {
				t.Errorf("expected id %s, got %s", session.ID(), got.ID())
			}

			if !got.Lifetime().Equal(session.Lifetime()) {
				t.Errorf("expected lifetime %v, got %v", session.Lifetime(), got.Lifetime())
			}

			if !got.Expiry().Equal(session.Expiry()) {
				t.Errorf("expected expiry %v, got %v", session.Expiry(), got.Expiry())
			}
		}
	})

	t.Run("DeleteSession", func(t *testing.T) {
		var sessionId string
		for _, session := range sess {
			err := manager.DeleteSession(t.Context(), session.ID())
			if err != nil {
				t.Fatalf("unexpected error on delte: %v", err)
			}
			sessionId = session.ID()
			break
		}

		_, ok, err := manager.GetSession(t.Context(), sessionId)
		if err != nil {
			t.Fatalf("unexpected error on get: %v", err)
		}

		if ok {
			t.Error("expected session to be deleted")
		}
	})

	t.Run("DeleteManySessions", func(t *testing.T) {
		err := manager.DeleteManySessions(t.Context(), slices.Collect(maps.Keys(sess))...)
		if err != nil {
			t.Fatalf("unexpected error on delte: %v", err)
		}

		for _, session := range sess {
			_, ok, err := manager.GetSession(t.Context(), session.ID())
			if err != nil {
				t.Fatalf("unexpected error on get: %v", err)
			}

			if ok {
				t.Error("expected session to be deleted")
			}
		}
	})

	t.Run("Commit nil", func(t *testing.T) {
		err := manager.Commit(t.Context(), nil)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	var sessionId string
	t.Run("Commit Default (StatusUnchanged, StatusChanged, StatusShouldRotate)", func(t *testing.T) {
		session := manager.NewSession()
		session.SetValue("test_key", 1234)
		sessionId = session.ID()

		err := manager.Commit(t.Context(), session)
		if err != nil {
			t.Fatalf("unexpected commit error: %v", err)
		}

		got, ok, err := manager.GetSession(t.Context(), sessionId)
		if err != nil {
			t.Fatalf("unexpected get error: %v", err)
		}

		if !ok {
			t.Fatal("session should have been returned from store")
		}

		if got.ID() != sessionId {
			t.Errorf("expected ID %s, got %s", sessionId, got.ID())
		}

		if value, ok := got.GetValue("test_key").(int); !ok || value != 1234 {
			t.Errorf("expected value 1234, got %d", value)
		}
	})

	t.Run("Commit Rotate (StatusRotate)", func(t *testing.T) {
		session, ok, err := manager.GetSession(t.Context(), sessionId)
		if err != nil {
			t.Fatalf("unexpected get error: %v", err)
		}

		if !ok {
			t.Fatalf("session should have been returned from store")
		}

		session.Rotate()

		err = manager.Commit(t.Context(), session)
		if err != nil {
			t.Fatalf("unexpected commit error: %v", err)
		}

		_, ok, err = manager.GetSession(t.Context(), sessionId)
		if err != nil {
			t.Fatalf("unexpected get error: %v", err)
		}

		if ok {
			t.Fatal("session should not have been returned from store")
		}
	})

	t.Run("Commit Destroy (StatusDestroy)", func(t *testing.T) {
		session := manager.NewSession()
		session.SetStatus(sessions.StatusChanged)

		err := manager.Commit(t.Context(), session)
		if err != nil {
			t.Fatalf("failed to commit initial session: %v", err)
		}

		_, ok, err := manager.GetSession(t.Context(), session.ID())
		if err != nil {
			t.Fatalf("unexpected get error: %v", err)
		}

		if !ok {
			t.Fatalf("session should have been returned from store")
		}

		session.Destroy()

		err = manager.Commit(t.Context(), session)
		if err != nil {
			t.Fatalf("unexpected commit error: %v", err)
		}

		_, ok, err = manager.GetSession(t.Context(), sessionId)
		if err != nil {
			t.Fatalf("unexpected get error: %v", err)
		}
		if ok {
			t.Error("session should have been deleted from store")
		}
	})
}
