package engine

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"sync"

	"github.com/amavis442/mockserver/internal/domain"
)

// Store is a thread-safe, in-memory collection of expectations. It is the
// sole source of truth for matching and administration.
type Store struct {
	mu   sync.RWMutex
	exps []domain.Expectation
}

// NewStore returns an empty Store.
func NewStore() *Store {
	return &Store{exps: make([]domain.Expectation, 0)}
}

// generateID returns a short random hex identifier.
func generateID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// sortedInsert inserts exp into the slice keeping it ordered by Priority
// descending. When Priority is equal, insertion order is preserved: new items
// come after existing items with the same priority.
func sortedInsert(entries []domain.Expectation, exp domain.Expectation) []domain.Expectation {
	idx := sort.Search(len(entries), func(i int) bool {
		return entries[i].Priority < exp.Priority
	})
	entries = append(entries, domain.Expectation{})
	copy(entries[idx+1:], entries[idx:])
	entries[idx] = exp
	return entries
}

// Upsert adds or replaces an expectation. If exp.ID is empty an id is
// generated automatically. When replacing, the Priority is updated and the
// entry is repositioned. Returns the stored expectation (with its id).
func (s *Store) Upsert(exp domain.Expectation) domain.Expectation {
	s.mu.Lock()
	defer s.mu.Unlock()

	if exp.ID == "" {
		exp.ID = generateID()
	}

	// Zero-value Times means "no explicit times set" — default to unlimited,
	// mirroring the JSON unmarshalling behaviour.
	if !exp.Times.Unlimited && exp.Times.Remaining == 0 {
		exp.Times.Unlimited = true
	}

	// Replace if exists.
	for i, e := range s.exps {
		if e.ID == exp.ID {
			s.exps = append(s.exps[:i], s.exps[i+1:]...)
			s.exps = sortedInsert(s.exps, exp)
			return exp
		}
	}

	s.exps = sortedInsert(s.exps, exp)
	return exp
}

// FindMatch scans expectations in order and returns the first available one
// whose RequestMatcher matches the given request. On match the expectation's
// Times are consumed, and if the expectation is exhausted it is removed from
// the store. Returns a copy so callers cannot mutate internal state. The
// second return value is false when no match is found.
func (s *Store) FindMatch(method, path string, headers map[string][]string) (domain.Expectation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.exps {
		exp := &s.exps[i]
		if !exp.Request.Match(method, path, headers) {
			continue
		}
		if !exp.Times.Available() {
			continue
		}
		exp.Times.Consume()
		cpy := *exp // shallow copy is safe — domain structs are value types
		if !exp.Times.Available() {
			// Remove exhausted expectation.
			s.exps = append(s.exps[:i], s.exps[i+1:]...)
		}
		return cpy, true
	}
	return domain.Expectation{}, false
}

// Remove deletes the expectation with the given id. It is a no-op when the id
// does not exist.
func (s *Store) Remove(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, e := range s.exps {
		if e.ID == id {
			s.exps = append(s.exps[:i], s.exps[i+1:]...)
			return true
		}
	}
	return false
}

// Reset removes all expectations.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exps = make([]domain.Expectation, 0)
}

// List returns a copy of the current expectations in priority order.
func (s *Store) List() []domain.Expectation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cp := make([]domain.Expectation, len(s.exps))
	copy(cp, s.exps)
	return cp
}
