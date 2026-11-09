package server

import (
	"cardfinders/internal/engine"
	"encoding/base64"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/argon2"
)

func testHashString(password string) string {
	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte(password), salt, 1, 64, 1, 16)
	enc := base64.RawStdEncoding
	return "$argon2id$v=19$m=64,t=1,p=1$" + enc.EncodeToString(salt) + "$" + enc.EncodeToString(key)
}

func authedHandler(t *testing.T) http.Handler {
	t.Helper()
	ph, err := ParsePasswordHash(testHashString("s3cret"))
	if err != nil {
		t.Fatal(err)
	}
	levels := []engine.Level{{Number: 1, Name: "T", SilverCount: 2, GoldCount: 1, StartLives: 5}}
	eng := engine.NewEngine(levels, rand.New(rand.NewSource(1)))
	return New(eng, Config{AuthHash: &ph}).Handler()
}

func do(h http.Handler, method, path, body, authz string) int {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestAuthProtectsLevelsAndCreate(t *testing.T) {
	h := authedHandler(t)
	for _, authz := range []string{"", "Bearer wrong", "s3cret", "Bearer "} {
		if c := do(h, "GET", "/api/levels", "", authz); c != http.StatusForbidden {
			t.Errorf("levels with %q: got %d, want 403", authz, c)
		}
		if c := do(h, "POST", "/api/game", `{"levelNumber":1}`, authz); c != http.StatusForbidden {
			t.Errorf("create with %q: got %d, want 403", authz, c)
		}
	}
	// Twice, so the second hit exercises the verdict cache.
	for range 2 {
		if c := do(h, "GET", "/api/levels", "", "Bearer s3cret"); c != http.StatusOK {
			t.Errorf("levels with good password: got %d", c)
		}
	}
	if c := do(h, "POST", "/api/game", `{"levelNumber":1}`, "Bearer s3cret"); c != http.StatusCreated {
		t.Errorf("create with good password: got %d", c)
	}
}

func TestJoinIsNotAuthProtected(t *testing.T) {
	h := authedHandler(t)
	if c := do(h, "POST", "/api/game/join", `{"joinCode":"nope"}`, ""); c == http.StatusForbidden {
		t.Errorf("join must not require auth, got 403")
	}
}

func TestParsePasswordHashRejectsGarbage(t *testing.T) {
	for _, s := range []string{"", "plain", "$argon2i$v=19$m=1,t=1,p=1$AA$AA", "$argon2id$v=19$m=0,t=1,p=1$AA$AA"} {
		if _, err := ParsePasswordHash(s); err == nil {
			t.Errorf("expected error for %q", s)
		}
	}
}
