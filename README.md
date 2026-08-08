# MockServer

A small, framework-agnostic HTTP mock server written in Go. Runs as a standalone
binary configurable via a JSON file at startup and/or an admin REST API at runtime,
so it can be used from any project (Go, Laravel, Symfony, C#/.NET, ...).

## Status

v2 — stable. Method/path/header matching, TLS, bearer token auth, refresh tokens, admin API, JSON config.

## Concept

An incoming request is matched against a list of *expectations*. The first matching
expectation wins and its configured response is returned. Expectations can be loaded
from a JSON config file at startup and/or managed at runtime via the admin API.

## Usage

```bash
mockserver --config expectations.json --port 8080
```

```bash
# Add an expectation at runtime
curl -X POST localhost:8080/__admin/expectations \
  -H "Content-Type: application/json" \
  -d '{"request":{"method":"GET","path":"/hello"},"response":{"status":200,"body":"world"}}'

curl localhost:8080/hello   # -> "world"
```

### TLS (HTTPS)

Use `--tls-self-signed` for local development — a certificate is generated in
memory at startup, valid for `localhost` for 365 days:

```bash
mockserver --tls-self-signed --port 8443
curl -k https://localhost:8443/hello   # -k skips certificate verification
```

Or provide your own certificate and key:

```bash
mockserver --tls-cert cert.pem --tls-key key.pem --port 8443
```

The server defaults to plain HTTP when no TLS flags are given.

### Auth flow (bearer tokens)

```bash
# Start with the example config (includes a protected endpoint /api/me)
mockserver --tls-self-signed --port 8443 --config testdata/example.json

# 1. Issue a token
TOKENS=$(curl -sk -X POST https://localhost:8443/__admin/auth/token \
  -H "Content-Type: application/json" \
  -d '{"subject":"pat"}')
ACCESS_TOKEN=$(echo "$TOKENS" | jq -r .token)
REFRESH_TOKEN=$(echo "$TOKENS" | jq -r .refresh_token)

# 2. Public endpoint — no auth needed
curl -sk https://localhost:8443/hello
# → {"message": "Hello from MockServer!"}

# 3. Protected endpoint — requires Bearer token
curl -sk -H "Authorization: Bearer $ACCESS_TOKEN" https://localhost:8443/api/me
# → {"id":1,"name":"Authenticated User"}

# 4. Without token → 401
curl -sk https://localhost:8443/api/me
# → {"error":"missing Authorization: Bearer"}

# 5. Refresh → old token revoked, new pair issued
REFRESHED=$(curl -sk -X POST https://localhost:8443/__admin/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")
NEW_TOKEN=$(echo "$REFRESHED" | jq -r .token)

# 6. Header-matched endpoint
curl -sk -H "Accept: application/json" https://localhost:8443/api/v2/users
# → {"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]}
```

## Expectation format

An expectation ties a request matcher to a response. The JSON shape is the same for
both the startup config file and the `POST /__admin/expectations` body:

```json
{
  "id": "unique-name",          // optional; auto-generated when omitted
  "priority": 100,              // higher = evaluated first; default 0
  "times": {                    // optional; defaults to unlimited (see below)
    "unlimited": true
  },
  "request": {
    "method": "GET",            // exact match
    "path": "/api/user",        // exact match
    "headers": {                // optional; when set, all must match (case-insensitive keys)
      "Authorization": "Bearer ***"
    }
  },
  "response": {
    "status": 200,              // HTTP status code
    "headers": {                // optional response headers
      "Content-Type": "application/json"
    },
    "body": { /* any JSON */ }  // optional; string, object, array, number, ...
  }
}
```

### The `times` field

`times` controls how often an expectation can be matched before it expires and
is automatically removed from the store.

**Allowed forms:**

| Form | Behaviour |
|---|---|
| Omitting `times` entirely | **Unlimited** — never expires. |
| `{"unlimited": true}` | Never expires. The `remaining` value is ignored. |
| `{"remaining": 3}` | Matches at most 3 times. Decrements on each hit; removed after the last. |
| `{"unlimited": true, "remaining": 5}` | Never expires (unlimited wins). |
| `{"remaining": 0}` | Immediately unavailable — skipped during matching. |

When both `unlimited` and `remaining` are present, `unlimited` takes precedence.
A missing `times` field is treated as `{"unlimited": true}`.

## Admin API

| Method | Path                          | Purpose                     |
|--------|-------------------------------|-----------------------------|
| GET    | `/__admin/expectations`       | List all expectations       |
| POST   | `/__admin/expectations`       | Add an expectation          |
| DELETE | `/__admin/expectations/{id}`  | Remove an expectation by id |
| POST   | `/__admin/reset`              | Remove all expectations     |

## License

[MIT](LICENSE.md)

## Acknowledgements

Inspired by [MockServer](https://github.com/mock-server/mockserver-monorepo) —
the mature, feature-rich Java mock server that defined the expectation/matcher/response
model and the `/__admin` API convention. This Go port borrows those core ideas while
staying intentionally minimal.
