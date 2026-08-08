# MockServer

A small, framework-agnostic HTTP mock server written in Go. Runs as a standalone
binary configurable via a JSON file at startup and/or an admin REST API at runtime,
so it can be used from any project (Go, Laravel, Symfony, C#/.NET, ...).

## Status

Work in progress — v1 (method + path matching, admin API, JSON config).

## Concept

An incoming request is matched against a list of *expectations*. The first matching
expectation wins and its configured response is returned. Expectations can be loaded
from a JSON config file at startup and/or managed at runtime via the admin API.

## Usage (planned)

```bash
mockserver --config expectations.json --port 8080
```

```bash
# Add an expectation at runtime
curl -X POST localhost:8080/__admin/expectations \
  -d '{"request":{"method":"GET","path":"/hello"},"response":{"status":200,"body":"world"}}'

curl localhost:8080/hello   # -> world
```

## Admin API

| Method | Path                          | Purpose                     |
|--------|-------------------------------|-----------------------------|
| GET    | `/__admin/expectations`       | List all expectations       |
| POST   | `/__admin/expectations`       | Add an expectation          |
| DELETE | `/__admin/expectations/{id}`  | Remove an expectation by id |
| POST   | `/__admin/reset`              | Remove all expectations     |

## License

TBD
