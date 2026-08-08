package domain

import "testing"

func TestRequestMatcher_Match(t *testing.T) {
	tests := []struct {
		name       string
		matcher    RequestMatcher
		method     string
		path       string
		reqHeaders map[string][]string
		want       bool
	}{
		{
			name:       "exact method and path match, no headers",
			matcher:    RequestMatcher{Method: "GET", Path: "/hello"},
			method:     "GET",
			path:       "/hello",
			reqHeaders: nil,
			want:       true,
		},
		{
			name:       "method mismatch",
			matcher:    RequestMatcher{Method: "GET", Path: "/hello"},
			method:     "POST",
			path:       "/hello",
			reqHeaders: nil,
			want:       false,
		},
		{
			name:       "path mismatch",
			matcher:    RequestMatcher{Method: "GET", Path: "/hello"},
			method:     "GET",
			path:       "/world",
			reqHeaders: nil,
			want:       false,
		},
		{
			name:       "both mismatch",
			matcher:    RequestMatcher{Method: "GET", Path: "/hello"},
			method:     "DELETE",
			path:       "/world",
			reqHeaders: nil,
			want:       false,
		},
		{
			name: "header match — single header",
			matcher: RequestMatcher{
				Method:  "GET",
				Path:    "/api",
				Headers: map[string]string{"Authorization": "Bearer abc"},
			},
			method:     "GET",
			path:       "/api",
			reqHeaders: map[string][]string{"Authorization": {"Bearer abc"}},
			want:       true,
		},
		{
			name: "header match — case-insensitive key",
			matcher: RequestMatcher{
				Method:  "GET",
				Path:    "/api",
				Headers: map[string]string{"Authorization": "Bearer abc"},
			},
			method:     "GET",
			path:       "/api",
			reqHeaders: map[string][]string{"authorization": {"Bearer abc"}},
			want:       true,
		},
		{
			name: "header mismatch — wrong value",
			matcher: RequestMatcher{
				Method:  "GET",
				Path:    "/api",
				Headers: map[string]string{"Authorization": "Bearer abc"},
			},
			method:     "GET",
			path:       "/api",
			reqHeaders: map[string][]string{"Authorization": {"Bearer wrong"}},
			want:       false,
		},
		{
			name: "header mismatch — missing header",
			matcher: RequestMatcher{
				Method:  "GET",
				Path:    "/api",
				Headers: map[string]string{"Authorization": "Bearer abc"},
			},
			method:     "GET",
			path:       "/api",
			reqHeaders: nil,
			want:       false,
		},
		{
			name: "multiple headers — all match",
			matcher: RequestMatcher{
				Method: "POST",
				Path:   "/data",
				Headers: map[string]string{
					"Content-Type": "application/json",
					"X-Api-Key":    "secret",
				},
			},
			method: "POST",
			path:   "/data",
			reqHeaders: map[string][]string{
				"Content-Type": {"application/json"},
				"X-Api-Key":    {"secret"},
			},
			want: true,
		},
		{
			name: "multiple headers — one mismatch",
			matcher: RequestMatcher{
				Method: "POST",
				Path:   "/data",
				Headers: map[string]string{
					"Content-Type": "application/json",
					"X-Api-Key":    "secret",
				},
			},
			method: "POST",
			path:   "/data",
			reqHeaders: map[string][]string{
				"Content-Type": {"application/json"},
				"X-Api-Key":    {"wrong"},
			},
			want: false,
		},
		{
			name: "matcher has no headers — always matches regardless of request headers",
			matcher: RequestMatcher{
				Method: "GET",
				Path:   "/public",
			},
			method:     "GET",
			path:       "/public",
			reqHeaders: map[string][]string{"Authorization": {"Bearer anything"}},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.matcher.Match(tt.method, tt.path, tt.reqHeaders); got != tt.want {
				t.Errorf("Match(%q, %q, %v) = %v, want %v", tt.method, tt.path, tt.reqHeaders, got, tt.want)
			}
		})
	}
}
