package auth_test

import (
	"regexp"
	"testing"

	"github.com/dimmerz92/quicky/auth"
)

func TestArgon2(t *testing.T) {
	var hash string
	test := "Pa55w0rd1!"
	pattern := regexp.MustCompile(`^\$argon2id\$v=\d+\$m=\d+,t=\d+,p=\d+\$[/+=A-z0-9]{24}\$[/+=A-z0-9+/]{44}$`)

	argon := auth.NewArgon2()

	t.Run("generate and compare", func(t *testing.T) {
		hash = argon.GenerateFromPassword(test)

		ok, err := argon.ComparePasswordAndHash(test, hash)
		if err != nil {
			t.Fatalf("failed to compare: %v", err)
		}

		if !pattern.MatchString(hash) {
			t.Fatalf("hash did not match expected pattern: %s", hash)
		}

		if !ok {
			t.Error("expected comparison to match")
		}
	})

	t.Run("unique", func(t *testing.T) {
		hash2 := argon.GenerateFromPassword(test)

		if hash == hash2 {
			t.Error("expected hashes to be different")
		}
	})
}
