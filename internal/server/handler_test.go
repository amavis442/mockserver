package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amavis442/mockserver/internal/domain"
	"github.com/amavis442/mockserver/internal/engine"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	store := engine.NewStore()
	h := NewHandler(store)
	return httptest.NewServer(h)
}

func expectJSON(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}

func addExpectation(t *testing.T, s *httptest.Server, exp domain.Expectation) domain.Expectation {
	t.Helper()
	body, _ := json.Marshal(exp)
	resp, err := s.Client().Post(s.URL+"/__admin/expectations", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST expectations: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST expectations: status %d, want 201", resp.StatusCode)
	}
	var created domain.Expectation
	expectJSON(t, resp, &created)
	return created
}

// mustGet is a test helper that calls s.Client().Get and fails on error.
func mustGet(t *testing.T, s *httptest.Server, path string) *http.Response {
	t.Helper()
	resp, err := s.Client().Get(s.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

// ── Admin API ──────────────────────────────────────────────────────────

func TestAdmin_GetExpectations_Empty(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	resp := mustGet(t, s, "/__admin/expectations")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var list []domain.Expectation
	expectJSON(t, resp, &list)
	if len(list) != 0 {
		t.Errorf("len = %d, want 0", len(list))
	}
}

func TestAdmin_PostAndGetExpectations(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	exp := domain.Expectation{
		Request:  domain.RequestMatcher{Method: "GET", Path: "/hello"},
		Response: domain.Response{Status: 200},
	}
	created := addExpectation(t, s, exp)

	if created.ID == "" {
		t.Error("expected auto-generated id")
	}
	if created.Request.Method != "GET" || created.Request.Path != "/hello" {
		t.Errorf("unexpected request matcher: %+v", created.Request)
	}

	// List should contain it
	resp := mustGet(t, s, "/__admin/expectations")
	defer resp.Body.Close()

	var list []domain.Expectation
	expectJSON(t, resp, &list)
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	if list[0].ID != created.ID {
		t.Errorf("listed id = %q, want %q", list[0].ID, created.ID)
	}
}

func TestAdmin_DeleteExpectation(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	exp := addExpectation(t, s, domain.Expectation{
		Request:  domain.RequestMatcher{Method: "GET", Path: "/x"},
		Response: domain.Response{Status: 200},
	})

	// Delete
	req, _ := http.NewRequest(http.MethodDelete, s.URL+"/__admin/expectations/"+exp.ID, nil)
	resp, err := s.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE status = %d, want 200", resp.StatusCode)
	}

	// List should be empty
	listResp := mustGet(t, s, "/__admin/expectations")
	defer listResp.Body.Close()
	var list []domain.Expectation
	json.NewDecoder(listResp.Body).Decode(&list)
	if len(list) != 0 {
		t.Errorf("len = %d, want 0", len(list))
	}
}

func TestAdmin_DeleteNonexistent(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	req, _ := http.NewRequest(http.MethodDelete, s.URL+"/__admin/expectations/nonexistent", nil)
	resp, err := s.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("DELETE nonexistent status = %d, want 404", resp.StatusCode)
	}
}

func TestAdmin_Reset(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	addExpectation(t, s, domain.Expectation{
		Request:  domain.RequestMatcher{Method: "GET", Path: "/a"},
		Response: domain.Response{Status: 200},
	})
	addExpectation(t, s, domain.Expectation{
		Request:  domain.RequestMatcher{Method: "GET", Path: "/b"},
		Response: domain.Response{Status: 200},
	})

	resp, err := s.Client().Post(s.URL+"/__admin/reset", "", nil)
	if err != nil {
		t.Fatalf("POST reset: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reset status = %d, want 200", resp.StatusCode)
	}

	// Should be empty now
	listResp := mustGet(t, s, "/__admin/expectations")
	defer listResp.Body.Close()
	var list []domain.Expectation
	json.NewDecoder(listResp.Body).Decode(&list)
	if len(list) != 0 {
		t.Errorf("len = %d after reset, want 0", len(list))
	}
}

// ── Mock matching ──────────────────────────────────────────────────────

func TestMock_MatchReturnsResponse(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	addExpectation(t, s, domain.Expectation{
		Request: domain.RequestMatcher{Method: "GET", Path: "/hello"},
		Response: domain.Response{
			Status:  201,
			Headers: map[string]string{"X-Custom": "hello-value"},
			Body:    json.RawMessage(`"world"`),
		},
	})

	resp := mustGet(t, s, "/hello")
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("status = %d, want 201", resp.StatusCode)
	}
	if h := resp.Header.Get("X-Custom"); h != "hello-value" {
		t.Errorf("X-Custom = %q, want hello-value", h)
	}
}

func TestMock_404WhenNoMatch(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	resp := mustGet(t, s, "/no-such-path")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}

	// Body should mention no expectation matched
	var body map[string]interface{}
	expectJSON(t, resp, &body)
	if msg, ok := body["error"].(string); !ok || msg == "" {
		t.Error("expected 'error' field in 404 body")
	}
}

func TestMock_ExpiresAfterLimit(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	addExpectation(t, s, domain.Expectation{
		Request: domain.RequestMatcher{Method: "GET", Path: "/once"},
		Response: domain.Response{
			Status: 200,
			Body:   json.RawMessage(`"hit"`),
		},
		Times: domain.Times{Remaining: 1},
	})

	// First hit — should match
	r1 := mustGet(t, s, "/once")
	r1.Body.Close()
	if r1.StatusCode != 200 {
		t.Errorf("first hit status = %d, want 200", r1.StatusCode)
	}

	// Second hit — should be 404 (expired)
	r2 := mustGet(t, s, "/once")
	r2.Body.Close()
	if r2.StatusCode != http.StatusNotFound {
		t.Errorf("second hit status = %d, want 404", r2.StatusCode)
	}
}

func TestMock_NotFoundIsJSON(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	resp := mustGet(t, s, "/__not-admin")
	defer resp.Body.Close()

	// Must be JSON, not HTML plaintext
	ct := resp.Header.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestMock_BodyIsServedRaw(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	addExpectation(t, s, domain.Expectation{
		Request: domain.RequestMatcher{Method: "GET", Path: "/raw"},
		Response: domain.Response{
			Status: 200,
			Body:   json.RawMessage(`{"key":"value"}`),
		},
	})

	resp := mustGet(t, s, "/raw")
	defer resp.Body.Close()

	var body map[string]string
	expectJSON(t, resp, &body)
	if body["key"] != "value" {
		t.Errorf("body = %v, want {key:value}", body)
	}
}

func TestMock_StringBodyIsServedAsPlain(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	addExpectation(t, s, domain.Expectation{
		Request: domain.RequestMatcher{Method: "GET", Path: "/text"},
		Response: domain.Response{
			Status: 200,
			Body:   json.RawMessage(`"just text"`),
		},
	})

	resp := mustGet(t, s, "/text")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

// ── Priority matching ──────────────────────────────────────────────────

func TestMock_FirstMatchWinsByPriority(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	addExpectation(t, s, domain.Expectation{
		ID:       "low",
		Priority: 10,
		Request:  domain.RequestMatcher{Method: "GET", Path: "/same"},
		Response: domain.Response{Status: 200, Body: json.RawMessage(`"low"`)},
	})
	addExpectation(t, s, domain.Expectation{
		ID:       "high",
		Priority: 100,
		Request:  domain.RequestMatcher{Method: "GET", Path: "/same"},
		Response: domain.Response{Status: 200, Body: json.RawMessage(`"high"`)},
	})

	resp := mustGet(t, s, "/same")
	defer resp.Body.Close()

	var body string
	json.NewDecoder(resp.Body).Decode(&body)
	// "high" was added second but has higher priority
	if body != "high" {
		t.Errorf("body = %q, want high", body)
	}
}

func TestMock_MethodMismatchIsNoMatch(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	addExpectation(t, s, domain.Expectation{
		Request:  domain.RequestMatcher{Method: "GET", Path: "/only-get"},
		Response: domain.Response{Status: 200},
	})

	resp, err := s.Client().Post(s.URL+"/only-get", "application/json", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("POST to GET-only matcher = %d, want 404", resp.StatusCode)
	}
}

// ── Admin / mock isolation ─────────────────────────────────────────────

func TestAdminRouteNotMatchedAsMock(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	// Add an expectation that would match /__admin/expectations if the
	// mock-router handled it, but admin routes take precedence.
	addExpectation(t, s, domain.Expectation{
		Request:  domain.RequestMatcher{Method: "GET", Path: "/__admin/expectations"},
		Response: domain.Response{Status: 418, Body: json.RawMessage(`"teapot"`)},
	})

	// Admin GET must NOT return the mock 418.
	resp := mustGet(t, s, "/__admin/expectations")
	defer resp.Body.Close()
	if resp.StatusCode == 418 {
		t.Error("admin route was caught by mock handler (should be real admin list)")
	}
}
