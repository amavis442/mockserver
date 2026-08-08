package auth

import (
	"testing"
	"time"
)

func TestIssue_ReturnsToken(t *testing.T) {
	store := NewTokenStore()
	info := store.Issue("user-1", 1*time.Hour)
	if info.Token == "" {
		t.Error("expected non-empty token")
	}
	if len(info.Token) < 32 {
		t.Errorf("token too short: len=%d", len(info.Token))
	}
	if info.Subject != "user-1" {
		t.Errorf("subject = %q, want user-1", info.Subject)
	}
}

func TestIssue_UniqueTokens(t *testing.T) {
	store := NewTokenStore()
	t1 := store.Issue("a", 1*time.Hour)
	t2 := store.Issue("a", 1*time.Hour)
	if t1.Token == t2.Token {
		t.Error("two tokens should be unique")
	}
}

func TestValidate_ValidToken(t *testing.T) {
	store := NewTokenStore()
	info := store.Issue("user-1", 1*time.Hour)

	got, ok := store.Validate(info.Token)
	if !ok {
		t.Fatal("expected valid token")
	}
	if got.Subject != "user-1" {
		t.Errorf("subject = %q, want user-1", got.Subject)
	}
}

func TestValidate_ExpiredToken(t *testing.T) {
	store := NewTokenStore()
	info := store.Issue("user-1", 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	_, ok := store.Validate(info.Token)
	if ok {
		t.Error("expired token should not validate")
	}
}

func TestValidate_RevokedToken(t *testing.T) {
	store := NewTokenStore()
	info := store.Issue("user-1", 1*time.Hour)
	store.Revoke(info.Token)

	_, ok := store.Validate(info.Token)
	if ok {
		t.Error("revoked token should not validate")
	}
}

func TestValidate_UnknownToken(t *testing.T) {
	store := NewTokenStore()
	_, ok := store.Validate("nonexistent-token")
	if ok {
		t.Error("unknown token should not validate")
	}
}

func TestRevokeAll(t *testing.T) {
	store := NewTokenStore()
	t1 := store.Issue("a", 1*time.Hour)
	t2 := store.Issue("b", 1*time.Hour)

	store.RevokeAll()

	if _, ok := store.Validate(t1.Token); ok {
		t.Error("t1 should be revoked")
	}
	if _, ok := store.Validate(t2.Token); ok {
		t.Error("t2 should be revoked")
	}
	if len(store.List()) != 0 {
		t.Errorf("list len = %d, want 0", len(store.List()))
	}
}

func TestList_ReturnsActiveTokens(t *testing.T) {
	store := NewTokenStore()
	store.Issue("a", 1*time.Hour)
	store.Issue("b", 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	list := store.List()
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1 (expired filtered out)", len(list))
	}
	if list[0].Subject != "a" {
		t.Errorf("subject = %q, want a", list[0].Subject)
	}
}

func TestIssue_ZeroTTLExpiresImmediately(t *testing.T) {
	store := NewTokenStore()
	info := store.Issue("x", 0)
	_, ok := store.Validate(info.Token)
	if ok {
		t.Error("token with zero TTL should not validate")
	}
}

// ── Refresh tokens ─────────────────────────────────────────────────────

func TestIssue_HasRefreshToken(t *testing.T) {
	store := NewTokenStore()
	info := store.Issue("u", 1*time.Hour)
	if info.RefreshToken == "" {
		t.Error("expected non-empty refresh_token")
	}
	if info.RefreshToken == info.Token {
		t.Error("refresh token should differ from access token")
	}
}

func TestRefresh_ReturnsNewToken(t *testing.T) {
	store := NewTokenStore()
	info := store.Issue("u", 1*time.Hour)

	newInfo, ok := store.Refresh(info.RefreshToken, 1*time.Hour)
	if !ok {
		t.Fatal("refresh should succeed")
	}
	if newInfo.Token == info.Token {
		t.Error("refreshed token should be new")
	}
	if newInfo.Subject != "u" {
		t.Errorf("subject = %q, want u", newInfo.Subject)
	}
}

func TestRefresh_RevokesOldToken(t *testing.T) {
	store := NewTokenStore()
	info := store.Issue("u", 1*time.Hour)

	store.Refresh(info.RefreshToken, 1*time.Hour)

	// Old access token should no longer validate.
	_, ok := store.Validate(info.Token)
	if ok {
		t.Error("old token should be revoked after refresh")
	}
}

func TestRefresh_NewTokenHasNewRefreshToken(t *testing.T) {
	store := NewTokenStore()
	info := store.Issue("u", 1*time.Hour)

	newInfo, _ := store.Refresh(info.RefreshToken, 1*time.Hour)
	if newInfo.RefreshToken == info.RefreshToken {
		t.Error("refresh token should be rotated too")
	}
}

func TestRefresh_WrongRefreshTokenFails(t *testing.T) {
	store := NewTokenStore()
	store.Issue("u", 1*time.Hour)

	_, ok := store.Refresh("wrong-refresh-token", 1*time.Hour)
	if ok {
		t.Error("refresh with unknown token should fail")
	}
}

func TestRefresh_ExpiredTokenCannotRefresh(t *testing.T) {
	store := NewTokenStore()
	info := store.Issue("u", 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	_, ok := store.Refresh(info.RefreshToken, 1*time.Hour)
	if ok {
		t.Error("refresh with expired token should fail")
	}
}
