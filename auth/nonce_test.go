package auth_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dimmerz92/quicky/auth"
)

func TestGenerateNonce(t *testing.T) {
	t.Run("length", func(t *testing.T) {
		n := auth.GenerateNonce(32)
		if len(n) != 32 {
			t.Fatalf("expected length 32, got %d", len(n))
		}
	})

	t.Run("unique", func(t *testing.T) {
		n1 := auth.GenerateNonce(32)
		n2 := auth.GenerateNonce(32)

		if bytes.Equal(n1, n2) {
			t.Fatal("expected different nonces, got identical values")
		}
	})
}

func TestGenerateURLSafeNonce(t *testing.T) {
	t.Run("no padding", func(t *testing.T) {
		s := auth.GenerateURLSafeNonce(32)

		if strings.Contains(s, "=") {
			t.Fatal("expected no padding '=' in URL-safe base64 string")
		}
	})

	t.Run("unique", func(t *testing.T) {
		s1 := auth.GenerateURLSafeNonce(32)
		s2 := auth.GenerateURLSafeNonce(32)

		if s1 == s2 {
			t.Fatal("expected different nonces, got identical values")
		}
	})
}
