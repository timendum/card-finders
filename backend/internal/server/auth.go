package server

// Bearer-password auth for the host-only endpoints (/api/levels, POST
// /api/game). The configured secret is an Argon2id hash in PHC string form;
// the client sends only the plain password.

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

const (
	// maxConcurrentVerifies bounds simultaneous Argon2 computations (memory).
	maxConcurrentVerifies = 2
	// maxCacheEntries bounds the verdict cache; it is cleared when full.
	maxCacheEntries = 256
)

// PasswordHash is a parsed Argon2id PHC string.
type PasswordHash struct {
	Memory  uint32 // KiB
	Time    uint32
	Threads uint8
	Salt    []byte
	Key     []byte
}

// Describe returns a log-safe summary (never the salt or key bytes).
func (h PasswordHash) Describe() string {
	return fmt.Sprintf("argon2id m=%dKiB t=%d p=%d saltLen=%d keyLen=%d",
		h.Memory, h.Time, h.Threads, len(h.Salt), len(h.Key))
}

// ParsePasswordHash parses "$argon2id$v=19$m=..,t=..,p=..$salt$key".
func ParsePasswordHash(s string) (PasswordHash, error) {
	var h PasswordHash
	parts := strings.Split(s, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return h, errors.New("expected $argon2id$v=19$m=..,t=..,p=..$salt$hash")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil ||
		version != argon2.Version {
		return h, fmt.Errorf("unsupported argon2 version %q", parts[2])
	}
	var threads uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d",
		&h.Memory, &h.Time, &threads); err != nil {
		return h, fmt.Errorf("invalid argon2 parameters %q", parts[3])
	}
	if h.Memory == 0 || h.Time == 0 || threads == 0 || threads > 255 {
		return h, errors.New("argon2 parameters must be positive (p <= 255)")
	}
	h.Threads = uint8(threads)
	enc := base64.RawStdEncoding
	var err error
	if h.Salt, err = enc.DecodeString(parts[4]); err != nil || len(h.Salt) == 0 {
		return h, errors.New("invalid argon2 salt encoding")
	}
	if h.Key, err = enc.DecodeString(parts[5]); err != nil || len(h.Key) == 0 {
		return h, errors.New("invalid argon2 hash encoding")
	}
	return h, nil
}

func (h PasswordHash) matches(password string) bool {
	got := argon2.IDKey([]byte(password), h.Salt, h.Time, h.Memory,
		h.Threads, uint32(len(h.Key)))
	return subtle.ConstantTimeCompare(got, h.Key) == 1
}

// authenticator verifies Bearer passwords against a PasswordHash. Order per
// request: verdict cache (keyed by SHA-256 of the password, never the
// password itself) -> semaphore -> Argon2id.
type authenticator struct {
	hash  PasswordHash
	sem   chan struct{}
	mu    sync.Mutex
	cache map[[32]byte]bool
}

func newAuthenticator(h PasswordHash) *authenticator {
	return &authenticator{
		hash:  h,
		sem:   make(chan struct{}, maxConcurrentVerifies),
		cache: make(map[[32]byte]bool),
	}
}

func (a *authenticator) check(r *http.Request) bool {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return false
	}
	password := h[len(prefix):]
	key := sha256.Sum256([]byte(password))

	a.mu.Lock()
	ok, hit := a.cache[key]
	a.mu.Unlock()
	if hit {
		return ok
	}

	select {
	case a.sem <- struct{}{}:
	case <-r.Context().Done():
		return false
	}
	ok = a.hash.matches(password)
	<-a.sem

	a.mu.Lock()
	if len(a.cache) >= maxCacheEntries {
		// clear cache, keep ok/true
		for key, value := range a.cache {
			if !value {
				delete(a.cache, key)
			}
		}
	}
	a.cache[key] = ok
	a.mu.Unlock()
	return ok
}

// requireAuth wraps next; with no authenticator configured it is a no-op.
// Any missing, malformed, or wrong credential yields 403 forbidden.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	if s.auth == nil {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.auth.check(r) {
			writeError(w, restError{http.StatusForbidden, "forbidden"},
				errors.New("invalid or missing credentials"))
			return
		}
		next(w, r)
	}
}
