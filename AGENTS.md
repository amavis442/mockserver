# AGENTS.md — MockServer

## Project goal
Standalone HTTP mock server in Go. An incoming request is matched against a list
of *expectations*; the first match wins and its configured response is returned.
Expectations come from a JSON config file at startup and/or are managed at
runtime via an admin REST API (`/__admin`). Framework-agnostic: usable from any
project (Go, Laravel, Symfony, C#/.NET) — it's just HTTP.

## Architecture (Clean Architecture / hexagonal)
```
cmd/mockserver/   — main binary, flags, config loading
internal/
  domain/         — pure structs, no dependencies
  engine/         — business logic (store, matching), no HTTP
  server/         — HTTP adapter (admin handler, mock handler)
```
No `pkg/` until there is genuinely reusable library code (YAGNI).

## Code style (predictability > everything)
- YAGNI, pragmatic, no over-engineering.
- One responsibility per file; code immediately understandable without docs.
- Mandatory: `.editorconfig` + clean `gofmt`/`go vet`.
- Stdlib first — only add a library if it saves significant boilerplate.

## Development rules
- **TDD**: write the test first (RED), watch it fail, then implement (GREEN).
- **Mocks, not real infra**: `httptest.NewServer` for HTTP tests, in-memory
  store for business logic. No Docker, no real network, no real database.
- **Commits**: atomic, conventional commits (`feat:`, `fix:`, `chore:`,
  `test:`), one thing per commit.

## Feature workflow (always follow)
1. Present a step plan for approval first.
2. Create a GitHub issue (problem + acceptance criteria) via `gh`.
3. Feature branch `feat/<n>-<description>` off `main`.
4. Step by step, minimal increments, only proceed after approval.
5. All tests green before committing.
6. Commit on feature branch → push → PR to `main` → then stop.

## Language
- Conversations: Dutch.
- Code, docs, commits, issues, PRs: English.

## Environment
- Go 1.25 (`/c/Program Files/Go/bin/go.exe` — not in PATH; export per session).
- Module: `github.com/amavis442/mockserver`.
- `gh` CLI: `/c/Program Files/GitHub CLI/gh.exe`.

## Admin API
| Method | Path                         | Purpose                     |
|--------|------------------------------|-----------------------------|
| GET    | `/__admin/expectations`      | List all expectations       |
| POST   | `/__admin/expectations`      | Add an expectation          |
| DELETE | `/__admin/expectations/{id}` | Remove an expectation by id |
| POST   | `/__admin/reset`             | Remove all expectations     |

## Out of scope (v1)
Query/header/body matchers, proxy, TLS, dashboards, record/playback, templating,
callbacks. They grow in later when a project genuinely needs them.
