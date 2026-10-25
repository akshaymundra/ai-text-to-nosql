# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Go service that turns natural-language requests into schema-grounded DynamoDB `QueryInput`s. Full design and Phase 1 scope live in `docs/idea.md`; read it before adding pipeline stages.

Repo layout: `server/` (this Go service) and `client/` (planned frontend for configuring schemas and running queries). Git root is the parent dir.

Core principle: the LLM is only a parser (English to IR). Index selection, scan detection, validation and compilation are deterministic code. The IR never names an index; the planner picks it. Keep LLM output out of any decision the planner/validator owns.

Pipeline (target, mostly unbuilt): schema retrieval (pgvector, one row per access pattern) → LLM intent extraction (Ollama structured output, enums from retrieved schema) → validator → dates (format check, widen pure-date bounds on timestamps) → access-path planner → compiler → `POST /api/v1/query`. Requests needing a scan are rejected; non-key conditions go to `FilterExpression` and return `valid` with a warning.

Build order from `docs/idea.md`: planner/validator/compiler are pure code, built test-first with no LLM, before retrieval and LLM work.

## Pipeline invariants

- Planner, validator and compiler are pure functions that take the tables as a parameter (`Plan(ir, tables)`), never call `dynamoschema.LoadDir` or read globals. Schemas will later come from a per-workspace registry.
- The IR never names an index. Only the planner sets `IndexName`.
- Rejections carry a structured code from `internal/constants` plus the offending field; human-readable explanations are built from those codes.
- Compiler output is deterministic: every attribute name and value is aliased (`#n0`, `:v0`, ... in condition order), so eval cases compare exactly.
- DynamoDB rules (key condition vs filter, GSI projection and consistency, date ranges, limits) live in the `dynamodb-query-rules` skill. Load it before touching IR operators, planner, validator or compiler.

## LLM and embeddings: Ollama only

- Use Ollama (local, or Ollama Cloud) for both intent extraction and embeddings. No other LLM provider; the `claude-api` skill does not apply here.
- Structured output (`format` = JSON schema on `/api/chat`) works with **local Ollama only**; Ollama Cloud does not support it (docs.ollama.com, checked 2026-10-07). With a cloud model, the IR must still be strictly parsed and validated, and a parse failure is a rejection, not a retry loop.
- Extraction calls use a pinned model tag, `temperature: 0` and a fixed `seed`.
- Embeddings via `/api/embed` (batch input, L2-normalised vectors, so cosine distance). Index and query must use the same embedding model; changing it means re-embedding every access pattern. Cloud availability of embedding models is undocumented, so embeddings run on local Ollama.

## Go conventions

- Logging is zap (`go.uber.org/zap`), not `slog`. No samber libraries. These override any community Go skill.
- Tests: stdlib `testing`, table-driven, with `github.com/google/go-cmp` for struct diffs (add the dependency when first needed). No testify.

## Commands

```sh
make run      # go run ./cmd/api (SERVER_PORT default 3000, APP_ENV default development)
make build    # bin/api
make test     # go test -race ./...
make lint     # golangci-lint run ./...
make seed     # go run ./cmd/seed-local
make register-demo   # needs the server running and DEMO_WORKSPACE_PASSWORD in .env; creates/logs in to workspace DEMO_WORKSPACE on SERVER_PORT and PUTs every schemas-local file (writes to Postgres: Akshay runs it)
make eval-retrieval  # read-only: retrieves tables for every eval case request in demo-local, prints recall@1 and recall@3
make migrate-create name=<snake_case_name>   # new goose migration in migrations/
make migrate-status                          # read-only
make migrate-up / make migrate-down          # Akshay only, never Claude
go test -race -run TestName ./internal/<pkg>   # single test
```

Migration targets read `DATABASE_URL` from `.env` via Make's `-include`, so a `$` or `#` in the URL must be escaped (`$$`, `\#`). Goose runs as a pinned tool (`go tool goose`, v3.27.0, the last release supporting Go 1.25).

`make seed` needs DynamoDB Local at `DYNAMODB_ENDPOINT` (default `http://localhost:8081`, region `DYNAMODB_REGION`). It **drops and recreates** every table in `schemas-local/`, then batch-inserts `exampleRecords`. Must run from `server/` (reads `schemas-local/` relatively).

## Architecture

- `schemas-local/*.json`: source of truth for tables. Each file defines table, PK/SK, attributes (with types and descriptions), GSIs, `accessPatterns` and `exampleRecords`. Loaded by `internal/dynamoschema.LoadDir`. Adding a table = adding a JSON file; seed and (later) retrieval pick it up automatically. Key attributes must be `string` or `number` (only those map to DynamoDB scalar types in `cmd/seed-local`).
- `cmd/api`: entrypoint. `App` wires config, zap logger, Postgres pool (pgx v5, `DATABASE_URL` required, pinged at startup) and chi router; graceful shutdown on SIGINT/SIGTERM closes the pool. Repos get the pool by injection and hold queries only.
- `migrations/`: goose SQL migrations for the Postgres app tables, named `<UTC timestamp>_<snake_name>.sql` (goose reads the version up to the first `_`). ERD and design notes in `docs/erd.md`. Create one file at a time: goose timestamps have 1s resolution, so two creates in the same second collide.
- `cmd/seed-local`: standalone DynamoDB Local seeder, uses static `local` credentials, loads `.env` itself, and refuses any endpoint whose host is not localhost/127.0.0.1/::1.
- `internal/routes`: `routes.go` sets global middleware (Recoverer, RequestID, request logger, CORS), `/health`, `/ready`, and mounts feature route groups in `registerRoutes`. Each feature gets its own file (e.g. `query.go`) that constructs service → handler and returns a `func(chi.Router)`.
- Layering: routes → `handler` → `service` → `repo`. API boilerplate is in place under `/api/v1`: `auth` (register, login; public), and behind `middleware.Authenticate`: `workspace` (get, `PUT /connection`), `schemas` (list, get/put/delete `/{tableName}`), `queries` (create, list, get `/{queryID}`, `POST /{queryID}/execute`, `GET /{queryID}/executions`). Handlers decode, call the service and write the response; service methods not built yet return `apperror.NotImplemented` (501).
- Auth: `POST /auth/register` and `/auth/login` take workspace `name` (stored lowercase) + password (bcrypt), return an HS256 JWT (`sub` = workspace ID, TTL `JWT_TTL`, secret `JWT_SECRET`, required). `middleware.Authenticate(authService, logger)` verifies `Authorization: Bearer <jwt>` via `AuthService.VerifyToken` and puts the workspace ID in the context for `middleware.WorkspaceID(ctx)`; handlers never take a workspace ID from the URL or body. Unknown name and wrong password return the same 401. Request payloads are validated by `validate` tags plus `req.Validate()` in `types/` (shared validator and custom tags in `types/validate.go`), called by the handler after decoding.
- Handlers decode bodies with `utils.DecodeJSON` (400 on bad JSON) and send every error through `utils.WriteError`: `*apperror.AppError` is written as-is, anything else is logged with `request_id` and returned as a generic 500.
- `internal/config`: `sync.Once` singleton from env vars; call `config.Init()` before `GetInstance()`. `Init` loads `.env` via godotenv (missing file ignored, real env vars win). Local tooling reads the same config; its keys (`DYNAMODB_ENDPOINT`, `DYNAMODB_REGION`, `DEMO_WORKSPACE`, `DEMO_WORKSPACE_PASSWORD`) have their own `.env.sample` section.
- `internal/apperror` + `internal/constants`: typed `AppError` with code and HTTP status; error codes live in `constants/errors.go`.
- `internal/utils/response.go`: `JSON` / `JSONError` wrap payloads as `{"data": ...}` / `{"error": ...}`; `WriteError` maps errors to responses. `utils/request.go`: `DecodeJSON`.
- `internal/model`: one file per Postgres table, structs with `db` tags matching `migrations/` (for `pgx.RowToStructByName`). Nullable columns are pointers or nil-able `json.RawMessage`; `jsonb` columns scan into typed fields (`dynamoschema.Table`, `[]string`). Column enums (`ConnectionType`, `QueryStatus`) live next to their model. A migration that changes a table must update its model.
- Empty placeholder dir (`routes/s2s`) marks where those concerns go. Pipeline stages go in `internal/service/<stage>/` (pure subpackages, wired by `QueryService`), external API wrappers in `internal/integrations/<name>/`. The full placement table is in the `go-conventions` skill; do not invent other packages.

## Project skills and hooks

- Skills in `server/.claude/skills/`: `go-conventions` (package layout, layering, errors, logging, tests, style; load before writing Go), `dynamodb-query-rules` (DynamoDB Query semantics, verified against AWS docs), `eval-harness` (eval case format, `make eval`, prompt-change workflow), `schema-authoring` (writing `schemas-local/*.json`).
- A Stop hook (`.claude/hooks/go-check.sh` at the repo root) runs `go vet ./...` and `go test ./...` in `server/` when Go files have uncommitted changes, and blocks once with the failure output. Fix failures; do not bypass it.
