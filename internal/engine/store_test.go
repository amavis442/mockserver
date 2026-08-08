package engine

import (
	"testing"

	"github.com/amavis442/mockserver/internal/domain"
)

func TestUpsert_GeneratesID(t *testing.T) {
	s := NewStore()
	exp := domain.Expectation{
		Request:  domain.RequestMatcher{Method: "GET", Path: "/"},
		Response: domain.Response{Status: 200},
	}
	result := s.Upsert(exp)
	if result.ID == "" {
		t.Error("expected auto-generated id")
	}
	if len(s.List()) != 1 {
		t.Fatalf("len = %d, want 1", len(s.List()))
	}
}

func TestUpsert_ReplacesByID(t *testing.T) {
	s := NewStore()
	exp := domain.Expectation{
		ID:       "abc",
		Request:  domain.RequestMatcher{Method: "GET", Path: "/a"},
		Response: domain.Response{Status: 200},
	}
	s.Upsert(exp)

	exp2 := domain.Expectation{
		ID:       "abc",
		Priority: 50,
		Request:  domain.RequestMatcher{Method: "POST", Path: "/b"},
		Response: domain.Response{Status: 201},
	}
	s.Upsert(exp2)

	list := s.List()
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1 (replaced)", len(list))
	}
	if list[0].Request.Method != "POST" {
		t.Errorf("method = %q, want POST", list[0].Request.Method)
	}
}

func TestUpsert_KeepsPriorityOrder(t *testing.T) {
	s := NewStore()
	s.Upsert(domain.Expectation{ID: "low", Priority: 10})
	s.Upsert(domain.Expectation{ID: "high", Priority: 100})
	s.Upsert(domain.Expectation{ID: "mid", Priority: 50})

	list := s.List()
	if len(list) != 3 {
		t.Fatal("expected 3 expectations")
	}
	want := []string{"high", "mid", "low"}
	for i, id := range want {
		if list[i].ID != id {
			t.Errorf("pos %d: id = %q, want %q", i, list[i].ID, id)
		}
	}
}

func TestUpsert_SamePriorityStable(t *testing.T) {
	s := NewStore()
	s.Upsert(domain.Expectation{ID: "first", Priority: 10})
	s.Upsert(domain.Expectation{ID: "second", Priority: 10})
	s.Upsert(domain.Expectation{ID: "third", Priority: 10})

	list := s.List()
	want := []string{"first", "second", "third"}
	for i, id := range want {
		if list[i].ID != id {
			t.Errorf("pos %d: id = %q, want %q", i, list[i].ID, id)
		}
	}
}

func TestFindMatch_BasicMatch(t *testing.T) {
	s := NewStore()
	s.Upsert(domain.Expectation{
		ID:       "echo",
		Request:  domain.RequestMatcher{Method: "GET", Path: "/hello"},
		Response: domain.Response{Status: 200},
	})

	_, ok := s.FindMatch("GET", "/hello", nil)
	if !ok {
		t.Fatal("expected match")
	}
}

func TestFindMatch_NoMatchReturnsFalse(t *testing.T) {
	s := NewStore()
	s.Upsert(domain.Expectation{
		Request:  domain.RequestMatcher{Method: "GET", Path: "/a"},
		Response: domain.Response{Status: 200},
	})

	_, ok := s.FindMatch("POST", "/a", nil)
	if ok {
		t.Error("expected no match for wrong method")
	}
	_, ok2 := s.FindMatch("GET", "/b", nil)
	if ok2 {
		t.Error("expected no match for wrong path")
	}
}

func TestFindMatch_ExpiresAfterLimit(t *testing.T) {
	s := NewStore()
	s.Upsert(domain.Expectation{
		ID:       "one-shot",
		Request:  domain.RequestMatcher{Method: "GET", Path: "/once"},
		Response: domain.Response{Status: 200},
		Times:    domain.Times{Remaining: 1},
	})

	// First hit — should match.
	_, ok := s.FindMatch("GET", "/once", nil)
	if !ok {
		t.Fatal("first match should succeed")
	}

	// Second hit — should miss (expired and removed).
	_, ok = s.FindMatch("GET", "/once", nil)
	if ok {
		t.Error("second match should fail after expiry")
	}

	// List should be empty.
	if len(s.List()) != 0 {
		t.Errorf("len = %d, want 0 after expiry", len(s.List()))
	}
}

func TestFindMatch_UnlimitedDoesNotExpire(t *testing.T) {
	s := NewStore()
	s.Upsert(domain.Expectation{
		ID:       "forever",
		Request:  domain.RequestMatcher{Method: "GET", Path: "/always"},
		Response: domain.Response{Status: 200},
		Times:    domain.Times{Unlimited: true},
	})

	for i := 0; i < 10; i++ {
		if _, ok := s.FindMatch("GET", "/always", nil); !ok {
			t.Fatalf("match #%d failed for unlimited expectation", i)
		}
	}
	if len(s.List()) != 1 {
		t.Error("unlimited expectation should not be removed")
	}
}

func TestFindMatch_FirstMatchWins(t *testing.T) {
	s := NewStore()
	s.Upsert(domain.Expectation{
		ID:       "high",
		Priority: 100,
		Request:  domain.RequestMatcher{Method: "GET", Path: "/x"},
		Response: domain.Response{Status: 200},
	})
	s.Upsert(domain.Expectation{
		ID:       "low",
		Priority: 10,
		Request:  domain.RequestMatcher{Method: "GET", Path: "/x"},
		Response: domain.Response{Status: 201},
	})

	exp, ok := s.FindMatch("GET", "/x", nil)
	if !ok {
		t.Fatal("expected match")
	}
	if exp.ID != "high" {
		t.Errorf("id = %q, want high (higher priority wins)", exp.ID)
	}
}

func TestFindMatch_HeaderMatching(t *testing.T) {
	s := NewStore()
	s.Upsert(domain.Expectation{
		ID: "auth-required",
		Request: domain.RequestMatcher{
			Method:  "GET",
			Path:    "/secure",
			Headers: map[string]string{"Authorization": "Bearer secret"},
		},
		Response: domain.Response{Status: 200},
	})

	// Match with correct header
	_, ok := s.FindMatch("GET", "/secure", map[string][]string{
		"Authorization": {"Bearer secret"},
	})
	if !ok {
		t.Error("expected match with correct header")
	}

	// No match with wrong header
	_, ok = s.FindMatch("GET", "/secure", map[string][]string{
		"Authorization": {"Bearer wrong"},
	})
	if ok {
		t.Error("expected no match with wrong header value")
	}

	// No match without header
	_, ok = s.FindMatch("GET", "/secure", nil)
	if ok {
		t.Error("expected no match without required header")
	}
}

func TestFindMatch_HeaderMatchingCaseInsensitive(t *testing.T) {
	s := NewStore()
	s.Upsert(domain.Expectation{
		ID: "case-test",
		Request: domain.RequestMatcher{
			Method:  "GET",
			Path:    "/x",
			Headers: map[string]string{"X-Custom": "val"},
		},
		Response: domain.Response{Status: 200},
	})

	// Match with different casing
	_, ok := s.FindMatch("GET", "/x", map[string][]string{
		"x-custom": {"val"},
	})
	if !ok {
		t.Error("expected match with different header casing")
	}
}

func TestRemove(t *testing.T) {
	s := NewStore()
	s.Upsert(domain.Expectation{ID: "keep"})
	s.Upsert(domain.Expectation{ID: "gone"})

	if !s.Remove("gone") {
		t.Error("Remove returned false for existing id")
	}
	if s.Remove("nonexistent") {
		t.Error("Remove returned true for nonexistent id")
	}

	list := s.List()
	if len(list) != 1 || list[0].ID != "keep" {
		t.Error("unexpected list after remove")
	}
}

func TestReset(t *testing.T) {
	s := NewStore()
	s.Upsert(domain.Expectation{ID: "a"})
	s.Upsert(domain.Expectation{ID: "b"})

	s.Reset()
	if len(s.List()) != 0 {
		t.Error("list not empty after reset")
	}
}

func TestList_ReturnsCopy(t *testing.T) {
	s := NewStore()
	s.Upsert(domain.Expectation{ID: "x"})

	list := s.List()
	list[0].ID = "mutated"

	actual := s.List()
	if actual[0].ID != "x" {
		t.Error("List returned a shared slice — mutation leaked")
	}
}

func TestFindMatch_ReturnsCopy(t *testing.T) {
	s := NewStore()
	s.Upsert(domain.Expectation{
		ID:       "original",
		Request:  domain.RequestMatcher{Method: "GET", Path: "/copy"},
		Response: domain.Response{Status: 200},
	})

	exp, ok := s.FindMatch("GET", "/copy", nil)
	if !ok {
		t.Fatal("no match")
	}
	exp.ID = "mutated"

	actual := s.List()
	if actual[0].ID != "original" {
		t.Error("FindMatch returned a shared struct — mutation leaked")
	}
}
