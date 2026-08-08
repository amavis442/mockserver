package domain

import "strings"

// RequestMatcher describes the conditions an incoming request must satisfy for
// an expectation to apply. Method and path are matched exactly. Headers, when
// set, require exact matches (case-insensitive key comparison); when nil or
// empty, headers are not checked.
type RequestMatcher struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
}

// Match reports whether the given request satisfies this matcher. Headers are
// optional — when the matcher has no headers configured, any request headers
// are accepted (backward compatible).
func (m RequestMatcher) Match(method, path string, reqHeaders map[string][]string) bool {
	if m.Method != method || m.Path != path {
		return false
	}

	if len(m.Headers) == 0 {
		return true // no header constraints — always match
	}

	for key, wantVal := range m.Headers {
		gotVal := headerValue(reqHeaders, key)
		if gotVal != wantVal {
			return false
		}
	}
	return true
}

// headerValue looks up a header key case-insensitively and returns the first
// value, or empty string if the header is not present.
func headerValue(headers map[string][]string, key string) string {
	if headers == nil {
		return ""
	}
	for k, v := range headers {
		if strings.EqualFold(k, key) && len(v) > 0 {
			return v[0]
		}
	}
	return ""
}
