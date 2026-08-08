package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// TokenInfo holds metadata about an issued token. Both the access token and
// its paired refresh token are included.
type TokenInfo struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Subject      string    `json:"subject"`
	IssuedAt     time.Time `json:"issued_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// TokenStore is a thread-safe, in-memory store for bearer tokens. Tokens are
// random hex strings — no real JWT cryptography is performed (that is the
// responsibility of the real server being mocked).
type TokenStore struct {
	mu            sync.RWMutex
	tokens        map[string]TokenInfo // access token → info
	refreshTokens map[string]string    // refresh token → access token
}

// NewTokenStore returns an empty TokenStore.
func NewTokenStore() *TokenStore {
	return &TokenStore{
		tokens:        make(map[string]TokenInfo),
		refreshTokens: make(map[string]string),
	}
}

// generateHex returns a random hex string of the given byte length.
func generateHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Issue generates a new access token with paired refresh token for the given
// subject with the specified TTL, stores both, and returns the TokenInfo.
func (s *TokenStore) Issue(subject string, ttl time.Duration) TokenInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	accessToken := generateHex(32)
	refreshToken := generateHex(32)

	now := time.Now()
	expiresAt := now.Add(ttl)
	if ttl <= 0 {
		expiresAt = now.Add(-1 * time.Second)
	}
	info := TokenInfo{
		Token:        accessToken,
		RefreshToken: refreshToken,
		Subject:      subject,
		IssuedAt:     now,
		ExpiresAt:    expiresAt,
	}
	s.tokens[accessToken] = info
	s.refreshTokens[refreshToken] = accessToken
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

// Refresh accepts a refresh token, revokes the current access token, and
// issues a new access token (with a new refresh token) for the same subject.
// The second return value is false when the refresh token is unknown or the
// underlying access token has expired.
func (s *TokenStore) Refresh(refreshToken string, ttl time.Duration) (TokenInfo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	accessToken, ok := s.refreshTokens[refreshToken]
	if !ok {
		return TokenInfo{}, false
	}
	info, ok := s.tokens[accessToken]
	if !ok {
		return TokenInfo{}, false
	}
	if time.Now().After(info.ExpiresAt) {
		return TokenInfo{}, false
	}

	// Revoke the old pair.
	delete(s.tokens, accessToken)
	delete(s.refreshTokens, refreshToken)

	// Issue a new pair.
	subject := info.Subject
	newAccess := generateHex(32)
	newRefresh := generateHex(32)
	now := time.Now()
	newInfo := TokenInfo{
		Token:        newAccess,
		RefreshToken: newRefresh,
		Subject:      subject,
		IssuedAt:     now,
		ExpiresAt:    now.Add(ttl),
	}
	s.tokens[newAccess] = newInfo
	s.refreshTokens[newRefresh] = newAccess
	return newInfo, true
}

// Revoke removes a single access token. It is a no-op if the token does not
// exist. The paired refresh token is also removed.
func (s *TokenStore) Revoke(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, ok := s.tokens[token]
	if ok {
		delete(s.refreshTokens, info.RefreshToken)
	}
	delete(s.tokens, token)
}

// RevokeAll removes all tokens and refresh tokens.
func (s *TokenStore) RevokeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens = make(map[string]TokenInfo)
	s.refreshTokens = make(map[string]string)
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
			delete(s.refreshTokens, info.RefreshToken)
			continue
		}
		result = append(result, info)
	}
	return result
}
