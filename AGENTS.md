# AGENTS.md — MockServer

## Projectdoel
Standalone HTTP mock server in Go. Een inkomend request wordt gematcht tegen een lijst
*expectations*; de eerste match wint en de bijbehorende response wordt teruggegeven.
Expectations komen uit een JSON-configbestand bij startup en/of worden runtime beheerd
via een admin REST API (`/__admin`). Framework-agnostisch: bruikbaar vanuit élk project
(Go, Laravel, Symfony, C#/.NET) — het is gewoon HTTP.

## Architectuur (Clean Architecture / hexagonaal)
```
cmd/mockserver/   — main binary, flags, config laden
internal/
  domain/         — pure structs, geen dependencies
  engine/         — businesslogica (store, matching), geen HTTP
  server/         — HTTP-adapter (admin handler, mock handler)
```
Geen `pkg/` tot er écht herbruikbare library-code is (YAGNI).

## Code-stijl (voorspelbaarheid > alles)
- YAGNI, pragmatisch, geen over-engineering.
- 1 verantwoordelijkheid per bestand; code meteen begrijpbaar zonder docs.
- Verplicht: `.editorconfig` + `gofmt`/`go vet` schoon.
- Stdlib first — alleen een library als die significant boilerplate bespaart.

## Development-regels
- **TDD**: eerst de test (RED), zien falen, dán implementeren (GREEN).
- **Mocks, geen echte infra**: `httptest.NewServer` voor HTTP-tests, in-memory store
  voor businesslogica. Geen Docker, geen echt netwerk, geen echte DB.
- **Commits**: atomair, conventional commits (`feat:`, `fix:`, `chore:`, `test:`),
  één ding per commit.

## Feature-workflow (altijd volgen)
1. Eerst stappenplan ter goedkeuring.
2. GitHub-issue aanmaken (probleem + acceptatiecriteria) via `gh`.
3. Feature branch `feat/<n>-<beschrijving>` vanaf `main`.
4. Stap voor stap, minimale increment, pas verder na ok.
5. Alle tests groen vóór commit.
6. Committen op feature branch → pushen → PR naar `main` → dan stop.

## Taal
- Gesprekken: Nederlands.
- Code, docs, commits, issues, PR's: Engels.

## Omgeving
- Go 1.25 (`/c/Program Files/Go/bin/go.exe` — niet in PATH; per sessie exporteren).
- Module: `github.com/amavis442/mockserver`.
- `gh` CLI: `/c/Program Files/GitHub CLI/gh.exe`.

## Admin API
| Method | Path                         | Doel                        |
|--------|------------------------------|-----------------------------|
| GET    | `/__admin/expectations`      | Lijst alle expectations     |
| POST   | `/__admin/expectations`      | Expectation toevoegen       |
| DELETE | `/__admin/expectations/{id}` | Expectation verwijderen     |
| POST   | `/__admin/reset`             | Alle expectations wissen    |

## Buiten scope (v1)
Query/header/body matchers, proxy, TLS, dashboards, record/playback, templating, callbacks.
Groeien er later in als een project ze écht nodig heeft.
