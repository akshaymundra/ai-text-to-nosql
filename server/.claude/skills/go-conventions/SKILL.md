---
name: go-conventions
description: Where Go code goes in this service and how it is written (package layout, layering, errors, logging, config, tests, style). Use before writing or reviewing any Go code in server/, adding an endpoint, service, repo, pipeline stage, external integration, env var or test.
license: MIT
metadata:
  adapted-from: samber/cc-skills-golang@golang-code-style v1.3.0 (MIT), trimmed and extended with this project's structure
---

# Go conventions for this service

Project rules beat community Go advice. "Clear is better than clever": minimal surface, no speculative abstraction, match the nearest existing file.

## 1. Where code goes

| Concern | Location | Pattern |
|---|---|---|
| Entrypoints | `cmd/api`, `cmd/seed-local` | `main` only wires things; no business logic. `cmd/api/app.go` opens DB clients (Postgres pool) and injects them downward. |
| Routes | `internal/routes/<feature>.go` | `func <Feature>Routes() func(chi.Router)` builds service → handler and mounts handlers; register it in `registerRoutes` in `routes.go`. |
| HTTP handlers | `internal/handler/<feature>.go` | Methods on the handler struct declared in `handler/init.go`. Decode request, call service, write response. No business logic. |
| Business logic | `internal/service/<feature>.go` | Methods on the service struct declared in `service/init.go`. Orchestrates pipeline stages, repos and integrations. |
| Persistence / external stores | `internal/repo/` | Queries only: DynamoDB execution, Postgres registry and pgvector queries. Receives its client (e.g. `*pgxpool.Pool`) via constructor; never opens connections. Returns domain types, not driver types. |
| Request/response DTOs | `internal/types/` | JSON-tagged structs for the HTTP API. |
| HTTP input validation | `internal/validations/` | Validates request payloads only. **Not** the DynamoDB query validator. |
| Persistence models | `internal/model/` | Structs mapped to Postgres rows. |
| Pipeline stages | `internal/service/<stage>/` | Subpackages of `service`: `ir`, `planner`, `validator`, `compiler` (later `retrieval`, `extraction`). Pure packages: input in, result out, tables passed as a parameter, no I/O, no globals, no logging. Only `service` (e.g. `QueryService`) wires them together. |
| Table schemas | `internal/schema/` | `schema.Table` and loaders. |
| External integrations | `internal/integrations/<name>/` | e.g. `integrations/ollama`. Thin typed wrapper over the external API; no pipeline logic. |
| Errors and codes | `internal/apperror/`, `internal/constants/` | See section 2. |
| Config | `internal/config/` | See section 4. |
| Cross-cutting HTTP helpers | `internal/utils/` | `JSON`, `JSONError`, `RequestLogger`. Do not grow this into a junk drawer. |
| Middleware | `internal/middleware/` | chi middleware only. |

Dependency direction: `routes → handler → service → (service/<stage>, repo, integrations) → schema/constants`. Stage packages import only other stage packages, `schema` and `constants`; never `service` itself, `repo`, `handler` or `integrations`. Nothing imports `cmd`.

Add a new top-level package only when no row above fits, and say why.

## 2. Errors

- Client-facing failures are `*apperror.AppError` built with the helpers (`apperror.BadRequest(...)`, `NotFound`, ...). Codes live in `internal/constants/errors.go`; add new codes there.
- Pipeline rejections (scan required, unknown field, bad sort-key operator) are structured results with a code from `internal/constants`, not Go errors. A rejected query is a normal outcome.
- Internal errors: wrap with context, `fmt.Errorf("loading schema %q: %w", name, err)`. Use `%q` for strings. Compare with `errors.Is` / `errors.As`.
- Handle an error once: either return it or log it, never both. Handlers are the boundary that logs and converts to a response.
- No `panic` outside `main` startup. An unexpected `switch` value returns an error.

## 3. Logging

- zap only (`*zap.Logger`, structured fields: `zap.String`, `zap.Error`). Never `slog`, `log`, or `fmt.Println` in `internal/`.
- The logger is injected (constructor parameter), not a package global. Stage packages do not log.
- Include `request_id` (`middleware.GetReqID(ctx)`) when logging inside a request.

## 4. Config and env

- New setting: add a field to `config.Config`, read it in `config.Init` with `getEnv(KEY, default)`, add the key to `server/.env.sample` (app keys only; seed/dev-only keys stay out).
- Code reads `config.GetInstance()` at wiring time and passes values down. Stage packages never read config or env.
- Secrets never get defaults in code.

## 5. Context and I/O

- `context.Context` is the first parameter of anything that does I/O (`repo`, `integrations`, `service`). Handlers pass `r.Context()`.
- Stage packages take no context; they do no I/O.
- Always set timeouts on outbound calls (Ollama, DynamoDB, Postgres).

## 6. Tests

- Stdlib `testing`, table-driven with `t.Run(tc.name, ...)`, diffs with `cmp.Diff` from `github.com/google/go-cmp/cmp`. No testify, no mocking frameworks.
- Test file next to the code (`planner_test.go`). Fixtures in the package's `testdata/`.
- Stage packages get exhaustive unit tests (see the checklist in `dynamodb-query-rules`). Fakes for `repo`/`integrations` are small hand-written structs implementing an interface declared by the consumer, only when a test needs one.
- `make test` runs with `-race`. A Stop hook runs `go vet` and `go test` automatically.

## 7. Style

- **Early return.** Handle errors and edge cases first; keep the happy path unindented. Drop `else` after `return`/`continue`/`break`.
- **Conditions with 3+ operands** become named booleans.
- **`switch`** instead of if-else chains on the same value.
- **Functions:** short, one job, at most 4 parameters (beyond that, an options struct). Order: `ctx`, inputs, outputs.
- **Declarations:** `:=` for non-zero values, `var` for zero values. Slices and maps that reach JSON are initialised (`[]T{}`), so they encode as `[]` not `null`. Preallocate only with a known size.
- **Composite literals** always use field names.
- **Files:** imports, constants, types, constructor, methods, helpers. One primary type per file when it has significant methods.
- **Surface:** unexport by default. No interface until a second implementation or a test fake needs it, and declare it where it is consumed.
- **Stdlib first:** `slices`, `maps`, `strings`, `strconv`. No `reflect`, no generic helper libraries, no dot imports, blank imports only in `main`.
- **Comments:** only for a non-obvious why (workaround, constraint, gotcha) or an exported API docblock where the package already uses them. Never restate code, label sections or leave TODOs.
- Lines past ~120 chars break at semantic boundaries; calls with many arguments go one per line.
- `gofmt`/`goimports` formatting; `make lint` runs golangci-lint.
