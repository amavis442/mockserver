package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/amavis442/mockserver/internal/auth"
	"github.com/amavis442/mockserver/internal/domain"
	"github.com/amavis442/mockserver/internal/engine"
)

const adminPrefix = "/__admin/"

// NewHandler returns an http.Handler that routes admin requests (under
// /__admin/) to the admin API and all other requests to the mock handler.
// The tokenStore may be nil — when nil, auth-required expectations will always
// fail with 401 (no token store available).
func NewHandler(store *engine.Store, tokenStore *auth.TokenStore) http.Handler {
	mux := http.NewServeMux()

	// Admin routes
	mux.HandleFunc("GET "+adminPrefix+"expectations", adminList(store))
	mux.HandleFunc("POST "+adminPrefix+"expectations", adminAdd(store))
	mux.HandleFunc("DELETE "+adminPrefix+"expectations/{id}", adminRemove(store))
	mux.HandleFunc("POST "+adminPrefix+"reset", adminReset(store))

	// Admin token routes (no-op when tokenStore is nil).
	mux.HandleFunc("POST "+adminPrefix+"auth/token", adminIssueToken(tokenStore))
	mux.HandleFunc("GET "+adminPrefix+"auth/tokens", adminListTokens(tokenStore))
	mux.HandleFunc("DELETE "+adminPrefix+"auth/tokens", adminRevokeAllTokens(tokenStore))
	mux.HandleFunc("POST "+adminPrefix+"auth/refresh", adminRefreshToken(tokenStore))

	// Mock catch-all
	mux.HandleFunc("/", mockHandler(store, tokenStore))

	return mux
}

// ── Admin handlers ─────────────────────────────────────────────────────

func adminList(store *engine.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list := store.List()
		writeJSON(w, http.StatusOK, list)
	}
}

func adminAdd(store *engine.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var exp domain.Expectation
		if err := json.NewDecoder(r.Body).Decode(&exp); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		created := store.Upsert(exp)
		writeJSON(w, http.StatusCreated, created)
	}
}

func adminRemove(store *engine.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !store.Remove(id) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "expectation not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
	}
}

func adminReset(store *engine.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.Reset()
		writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
	}
}

// ── Token admin handlers ───────────────────────────────────────────────

func adminIssueToken(ts *auth.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ts == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "token store not available"})
			return
		}
		var req struct {
			Subject string `json:"subject"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if req.Subject == "" {
			req.Subject = "default"
		}
		info := ts.Issue(req.Subject, 1*time.Hour)
		writeJSON(w, http.StatusCreated, info)
	}
}

func adminListTokens(ts *auth.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ts == nil {
			writeJSON(w, http.StatusOK, []auth.TokenInfo{})
			return
		}
		writeJSON(w, http.StatusOK, ts.List())
	}
}

func adminRevokeAllTokens(ts *auth.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ts != nil {
			ts.RevokeAll()
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "tokens revoked"})
	}
}

func adminRefreshToken(ts *auth.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ts == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "token store not available"})
			return
		}
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		newInfo, ok := ts.Refresh(req.RefreshToken, 1*time.Hour)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired refresh token"})
			return
		}
		writeJSON(w, http.StatusCreated, newInfo)
	}
}

// ── Mock handler ───────────────────────────────────────────────────────

func mockHandler(store *engine.Store, tokenStore *auth.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Admin paths should never reach here (ServeMux dispatches them
		// first), but guard defensively regardless.
		if strings.HasPrefix(r.URL.Path, adminPrefix) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}

		exp, ok := store.FindMatch(r.Method, r.URL.Path, r.Header)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":  "no expectation matched",
				"method": r.Method,
				"path":   r.URL.Path,
			})
			return
		}

		// Auth check.
		if exp.Auth != nil && exp.Auth.Required {
			if tokenStore == nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "auth required but no token store configured"})
				return
			}
			token := bearerToken(r)
			if token == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing Authorization: Bearer token"})
				return
			}
			if _, valid := tokenStore.Validate(token); !valid {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
				return
			}
		}

		// Response headers.
		for k, v := range exp.Response.Headers {
			w.Header().Set(k, v)
		}

		status := exp.Response.Status
		if status == 0 {
			status = http.StatusOK
		}

		// Body.
		if exp.Response.Body != nil {
			// Determine Content-Type: if the caller set one via Headers,
			// keep it; otherwise default to application/json.
			if w.Header().Get("Content-Type") == "" {
				w.Header().Set("Content-Type", "application/json")
			}
			w.WriteHeader(status)
			w.Write(exp.Response.Body)
			return
		}

		w.WriteHeader(status)
	}
}

// bearerToken extracts the token from an Authorization: Bearer header.
func bearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(auth) > len(prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
		return auth[len(prefix):]
	}
	return ""
}

// ── Helpers ─────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
