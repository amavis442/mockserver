package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amavis442/mockserver/internal/auth"
	"github.com/amavis442/mockserver/internal/domain"
	"github.com/amavis442/mockserver/internal/engine"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	store := engine.NewStore()
	tokenStore := auth.NewTokenStore()
	h := NewHandler(store, tokenStore)
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

	req, _ := http.NewRequest(http.MethodDelete, s.URL+"/__admin/expectations/"+exp.ID, nil)
	resp, err := s.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE status = %d, want 200", resp.StatusCode)
	}

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

	r1 := mustGet(t, s, "/once")
	r1.Body.Close()
	if r1.StatusCode != 200 {
		t.Errorf("first hit status = %d, want 200", r1.StatusCode)
	}

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

	addExpectation(t, s, domain.Expectation{
		Request:  domain.RequestMatcher{Method: "GET", Path: "/__admin/expectations"},
		Response: domain.Response{Status: 418, Body: json.RawMessage(`"teapot"`)},
	})

	resp := mustGet(t, s, "/__admin/expectations")
	defer resp.Body.Close()
	if resp.StatusCode == 418 {
		t.Error("admin route was caught by mock handler (should be real admin list)")
	}
}

// ── Header matching ────────────────────────────────────────────────────

func TestMock_HeaderMatching(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	addExpectation(t, s, domain.Expectation{
		Request: domain.RequestMatcher{
			Method:  "GET",
			Path:    "/secure",
			Headers: map[string]string{"Authorization": "Bearer secret"},
		},
		Response: domain.Response{Status: 200, Body: json.RawMessage(`"ok"`)},
	})

	req, _ := http.NewRequest(http.MethodGet, s.URL+"/secure", nil)
	req.Header.Set("Authorization", "Bearer secret")
	resp, err := s.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /secure: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	req2, _ := http.NewRequest(http.MethodGet, s.URL+"/secure", nil)
	req2.Header.Set("Authorization", "Bearer wrong")
	resp2, err := s.Client().Do(req2)
	if err != nil {
		t.Fatalf("GET /secure (wrong header): %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("wrong header: status = %d, want 404", resp2.StatusCode)
	}

	resp3 := mustGet(t, s, "/secure")
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusNotFound {
		t.Errorf("missing header: status = %d, want 404", resp3.StatusCode)
	}
}

func TestMock_HeaderMatchingCaseInsensitive(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	addExpectation(t, s, domain.Expectation{
		Request: domain.RequestMatcher{
			Method:  "GET",
			Path:    "/x",
			Headers: map[string]string{"X-Custom": "val"},
		},
		Response: domain.Response{Status: 200, Body: json.RawMessage(`"ok"`)},
	})

	req, _ := http.NewRequest(http.MethodGet, s.URL+"/x", nil)
	req.Header.Set("x-custom", "val")
	resp, err := s.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /x: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

// ── Auth ────────────────────────────────────────────────────────────────

func TestAuth_TokenAdminAPI(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	resp, err := s.Client().Post(s.URL+"/__admin/auth/token", "application/json",
		bytes.NewReader([]byte(`{"subject":"user-1"}`)))
	if err != nil {
		t.Fatalf("POST token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("issue token status = %d, want 201", resp.StatusCode)
	}

	var info auth.TokenInfo
	expectJSON(t, resp, &info)
	if info.Token == "" {
		t.Error("expected token in response")
	}
	if info.Subject != "user-1" {
		t.Errorf("subject = %q, want user-1", info.Subject)
	}

	listResp := mustGet(t, s, "/__admin/auth/tokens")
	defer listResp.Body.Close()
	var tokens []auth.TokenInfo
	json.NewDecoder(listResp.Body).Decode(&tokens)
	if len(tokens) != 1 {
		t.Fatalf("token count = %d, want 1", len(tokens))
	}

	revokeReq, _ := http.NewRequest(http.MethodDelete, s.URL+"/__admin/auth/tokens", nil)
	revokeResp, _ := s.Client().Do(revokeReq)
	revokeResp.Body.Close()

	emptyResp := mustGet(t, s, "/__admin/auth/tokens")
	defer emptyResp.Body.Close()
	var empty []auth.TokenInfo
	json.NewDecoder(emptyResp.Body).Decode(&empty)
	if len(empty) != 0 {
		t.Errorf("token count after revoke = %d, want 0", len(empty))
	}
}

func TestAuth_Required_AllowsValidToken(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	resp, err := s.Client().Post(s.URL+"/__admin/auth/token", "application/json",
		bytes.NewReader([]byte(`{"subject":"u1"}`)))
	if err != nil {
		t.Fatalf("POST token: %v", err)
	}
	defer resp.Body.Close()
	var info auth.TokenInfo
	json.NewDecoder(resp.Body).Decode(&info)

	addExpectation(t, s, domain.Expectation{
		Request:  domain.RequestMatcher{Method: "GET", Path: "/secure"},
		Response: domain.Response{Status: 200, Body: json.RawMessage(`"secret"`)},
		Auth:     &domain.AuthConfig{Required: true},
	})

	req, _ := http.NewRequest(http.MethodGet, s.URL+"/secure", nil)
	req.Header.Set("Authorization", "Bearer "+info.Token)
	r, err := s.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /secure: %v", err)
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		t.Errorf("status = %d, want 200", r.StatusCode)
	}
}

func TestAuth_Required_RejectsMissingToken(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	addExpectation(t, s, domain.Expectation{
		Request:  domain.RequestMatcher{Method: "GET", Path: "/secure"},
		Response: domain.Response{Status: 200},
		Auth:     &domain.AuthConfig{Required: true},
	})

	resp := mustGet(t, s, "/secure")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestAuth_Required_RejectsInvalidToken(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	addExpectation(t, s, domain.Expectation{
		Request:  domain.RequestMatcher{Method: "GET", Path: "/secure"},
		Response: domain.Response{Status: 200},
		Auth:     &domain.AuthConfig{Required: true},
	})

	req, _ := http.NewRequest(http.MethodGet, s.URL+"/secure", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	r, err := s.Client().Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", r.StatusCode)
	}
}

func TestAuth_NotRequired_AllowsWithoutToken(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	addExpectation(t, s, domain.Expectation{
		Request:  domain.RequestMatcher{Method: "GET", Path: "/public"},
		Response: domain.Response{Status: 200, Body: json.RawMessage(`"public"`)},
	})

	resp := mustGet(t, s, "/public")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

// ── Refresh token ───────────────────────────────────────────────────────

func TestAuth_RefreshToken(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	resp, err := s.Client().Post(s.URL+"/__admin/auth/token", "application/json",
		bytes.NewReader([]byte(`{"subject":"u1"}`)))
	if err != nil {
		t.Fatalf("POST token: %v", err)
	}
	defer resp.Body.Close()
	var info auth.TokenInfo
	json.NewDecoder(resp.Body).Decode(&info)
	if info.RefreshToken == "" {
		t.Fatal("expected refresh_token in response")
	}

	refreshBody := `{"refresh_token":"` + info.RefreshToken + `"}`
	refreshResp, err := s.Client().Post(s.URL+"/__admin/auth/refresh", "application/json",
		bytes.NewReader([]byte(refreshBody)))
	if err != nil {
		t.Fatalf("POST refresh: %v", err)
	}
	defer refreshResp.Body.Close()
	if refreshResp.StatusCode != http.StatusCreated {
		t.Fatalf("refresh status = %d, want 201", refreshResp.StatusCode)
	}

	var newInfo auth.TokenInfo
	json.NewDecoder(refreshResp.Body).Decode(&newInfo)
	if newInfo.Token == info.Token {
		t.Error("refreshed access token should be new")
	}
	if newInfo.RefreshToken == info.RefreshToken {
		t.Error("refresh token should be rotated")
	}

	addExpectation(t, s, domain.Expectation{
		Request:  domain.RequestMatcher{Method: "GET", Path: "/secure"},
		Response: domain.Response{Status: 200},
		Auth:     &domain.AuthConfig{Required: true},
	})

	// Old token rejected.
	req, _ := http.NewRequest(http.MethodGet, s.URL+"/secure", nil)
	req.Header.Set("Authorization", "Bearer "+info.Token)
	r, _ := s.Client().Do(req)
	r.Body.Close()
	if r.StatusCode != http.StatusUnauthorized {
		t.Errorf("old token status = %d, want 401", r.StatusCode)
	}

	// New token accepted.
	req2, _ := http.NewRequest(http.MethodGet, s.URL+"/secure", nil)
	req2.Header.Set("Authorization", "Bearer "+newInfo.Token)
	r2, err := s.Client().Do(req2)
	if err != nil {
		t.Fatalf("GET /secure with new token: %v", err)
	}
	defer r2.Body.Close()
	if r2.StatusCode != 200 {
		t.Errorf("new token after refresh: status = %d, want 200", r2.StatusCode)
	}
}
