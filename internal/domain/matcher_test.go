package domain

import "testing"

func TestRequestMatcher_Match(t *testing.T) {
	tests := []struct {
		name    string
		matcher RequestMatcher
		method  string
		path    string
		want    bool
	}{
		{
			name:    "exact method and path match",
			matcher: RequestMatcher{Method: "GET", Path: "/hello"},
			method:  "GET",
			path:    "/hello",
			want:    true,
		},
		{
			name:    "method mismatch",
			matcher: RequestMatcher{Method: "GET", Path: "/hello"},
			method:  "POST",
			path:    "/hello",
			want:    false,
		},
		{
			name:    "path mismatch",
			matcher: RequestMatcher{Method: "GET", Path: "/hello"},
			method:  "GET",
			path:    "/world",
			want:    false,
		},
		{
			name:    "both mismatch",
			matcher: RequestMatcher{Method: "GET", Path: "/hello"},
			method:  "DELETE",
			path:    "/world",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.matcher.Match(tt.method, tt.path); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.method, tt.path, got, tt.want)
			}
		})
	}
}
