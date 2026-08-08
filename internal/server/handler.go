package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/amavis442/mockserver/internal/domain"
	"github.com/amavis442/mockserver/internal/engine"
)

const adminPrefix = "/__admin/"

// NewHandler returns an http.Handler that routes admin requests (under
// /__admin/) to the admin API and all other requests to the mock handler.
func NewHandler(store *engine.Store) http.Handler {
	mux := http.NewServeMux()

	// Admin routes
	mux.HandleFunc("GET "+adminPrefix+"expectations", adminList(store))
	mux.HandleFunc("POST "+adminPrefix+"expectations", adminAdd(store))
	mux.HandleFunc("DELETE "+adminPrefix+"expectations/{id}", adminRemove(store))
	mux.HandleFunc("POST "+adminPrefix+"reset", adminReset(store))

	// Mock catch-all
	mux.HandleFunc("/", mockHandler(store))

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

// ── Mock handler ───────────────────────────────────────────────────────

func mockHandler(store *engine.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Admin paths should never reach here (ServeMux dispatches them
		// first), but guard defensively regardless.
		if strings.HasPrefix(r.URL.Path, adminPrefix) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}

		exp, ok := store.FindMatch(r.Method, r.URL.Path)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":  "no expectation matched",
				"method": r.Method,
				"path":   r.URL.Path,
			})
			return
		}

		// Response headers.
		for k, v := range exp.Response.Headers {
			w.Header().Set(k, v)
		}

		// Body.
		if exp.Response.Body != nil {
			// Determine Content-Type: if the caller set one via Headers,
			// keep it; otherwise default to application/json.
			if w.Header().Get("Content-Type") == "" {
				w.Header().Set("Content-Type", "application/json")
			}
			w.WriteHeader(exp.Response.Status)
			w.Write(exp.Response.Body)
			return
		}

		w.WriteHeader(exp.Response.Status)
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
