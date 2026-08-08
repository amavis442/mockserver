package domain

// RequestMatcher describes the conditions an incoming request must satisfy for
// an expectation to apply. In v1 only exact method and path matching are
// supported; additional matchers (query, headers, body) may be added later.
type RequestMatcher struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// Match reports whether the given request method and path satisfy this matcher.
// Both must match exactly.
func (m RequestMatcher) Match(method, path string) bool {
	return m.Method == method && m.Path == path
}
