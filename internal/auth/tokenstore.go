package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// TokenInfo holds metadata about an issued token.
type TokenInfo struct {
	Token     string    `json:"token"`
	Subject   string    `json:"subject"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// TokenStore is a thread-safe, in-memory store for bearer tokens. Tokens are
// random hex strings — no real JWT cryptography is performed (that is the
// responsibility of the real server being mocked).
type TokenStore struct {
	mu     sync.RWMutex
	tokens map[string]TokenInfo
}

// NewTokenStore returns an empty TokenStore.
func NewTokenStore() *TokenStore {
	return &TokenStore{tokens: make(map[string]TokenInfo)}
}

// Issue generates a new token for the given subject with the specified TTL,
// stores it, and returns its TokenInfo.
func (s *TokenStore) Issue(subject string, ttl time.Duration) TokenInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)

	now := time.Now()
	expiresAt := now.Add(ttl)
	if ttl <= 0 {
		// Ensure immediate expiry even when TTL rounds to zero.
		expiresAt = now.Add(-1 * time.Second)
	}
	info := TokenInfo{
		Token:     token,
		Subject:   subject,
		IssuedAt:  now,
		ExpiresAt: expiresAt,
	}
	s.tokens[token] = info
	return info
}

// Validate checks whether a token exists and has not expired. Returns the
// TokenInfo and true when valid, or zero-value and false otherwise.
func (s *TokenStore) Validate(token string) (TokenInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, ok := s.tokens[token]
	if !ok {
		return TokenInfo{}, false
	}
	if time.Now().After(info.ExpiresAt) {
		return TokenInfo{}, false
	}
	return info, true
}

// Revoke removes a single token. It is a no-op if the token does not exist.
func (s *TokenStore) Revoke(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, token)
}

// RevokeAll removes all tokens.
func (s *TokenStore) RevokeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens = make(map[string]TokenInfo)
}

// List returns all currently valid (non-expired) tokens. Expired tokens are
// filtered out and cleaned up from the store.
func (s *TokenStore) List() []TokenInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	result := make([]TokenInfo, 0, len(s.tokens))
	for token, info := range s.tokens {
		if now.After(info.ExpiresAt) {
			delete(s.tokens, token)
			continue
		}
		result = append(result, info)
	}
	return result
}
