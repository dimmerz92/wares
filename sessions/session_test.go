package sessions_test

import (
	"testing"
	"time"

	"github.com/dimmerz92/wares/sessions"
)

func TestSession(t *testing.T) {
	key := "test_key"
	value := 1234
	lifetime := time.Now().Add(50 * time.Millisecond)
	session := sessions.NewSession(lifetime, lifetime)

	t.Run("SetValue and GetValue", func(t *testing.T) {
		if got := session.GetStatus(); got != sessions.StatusUnchanged {
			t.Fatalf("expected initial status %d, got %d", sessions.StatusUnchanged, got)
		}

		session.SetValue(key, value)

		if got, ok := session.GetValue(key).(int); !ok {
			t.Fatal("expected value")
		} else if got != value {
			t.Errorf("expected %d, got %d", value, got)
		}

		if got := session.GetStatus(); got != sessions.StatusChanged {
			t.Fatalf("expected initial status %d, got %d", sessions.StatusChanged, got)
		}
	})

	t.Run("DeleteValue", func(t *testing.T) {
		session.DeleteValue(key)

		if _, ok := session.GetValue(key).(int); ok {
			t.Fatal("expected value")
		}
	})

	t.Run("Rotate", func(t *testing.T) {
		session.Rotate()

		if got := session.GetStatus(); got != sessions.StatusRotate {
			t.Fatalf("expected initial status %d, got %d", sessions.StatusRotate, got)
		}
	})

	t.Run("Destroy", func(t *testing.T) {
		session.Destroy()

		session.SetValue(key, value)

		if _, ok := session.GetValue(key).(int); ok {
			t.Fatal("unexpected value")
		}

		if got := session.GetStatus(); got != sessions.StatusDestroy {
			t.Fatalf("expected initial status %d, got %d", sessions.StatusDestroy, got)
		}
	})
}
