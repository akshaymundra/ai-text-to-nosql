# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Go service that turns natural-language requests into schema-grounded DynamoDB `QueryInput`s. Full design and Phase 1 scope live in `docs/idea.md`; read it before adding pipeline stages.

Repo layout: `server/` (this Go service) and `client/` (planned frontend for configuring schemas and running queries). Git root is the parent dir.

Core principle: the LLM is only a parser (English to IR). Index selection, scan detection, validation and compilation are deterministic code. The IR never names an index; the planner picks it. Keep LLM output out of any decision the planner/validator owns.

Pipeline (target, mostly unbuilt): schema retrieval (pgvector, one row per access pattern) → LLM intent extraction (Ollama structured output, enums from retrieved schema) → access-path planner → validator → compiler → `POST /api/v1/query`. Requests needing a scan are rejected; non-key conditions go to `FilterExpression` and return `valid` with a warning.

Build order from `docs/idea.md`: planner/validator/compiler are pure code, built test-first with no LLM, before retrieval and LLM work.

## Pipeline invariants

- Planner, validator and compiler are pure functions that take the tables as a parameter (`Plan(ir, tables)`), never call `schema.LoadDir` or read globals. Schemas will later come from a per-workspace registry.
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
go test -race -run TestName ./internal/<pkg>   # single test
```

`make seed` needs DynamoDB Local (default `http://localhost:8081`, override with `DYNAMODB_ENDPOINT`). It **drops and recreates** every table in `schemas-local/`, then batch-inserts `exampleRecords`. Must run from `server/` (reads `schemas-local/` relatively).

## Architecture

- `schemas-local/*.json`: source of truth for tables. Each file defines table, PK/SK, attributes (with types and descriptions), GSIs, `accessPatterns` and `exampleRecords`. Loaded by `internal/schema.LoadDir`. Adding a table = adding a JSON file; seed and (later) retrieval pick it up automatically. Key attributes must be `string` or `number` (only those map to DynamoDB scalar types in `cmd/seed-local`).
- `cmd/api`: entrypoint. `App` wires config, zap logger, Postgres pool (pgx v5, `DATABASE_URL` required, pinged at startup) and chi router; graceful shutdown on SIGINT/SIGTERM closes the pool. Repos get the pool by injection and hold queries only.
- `cmd/seed-local`: standalone DynamoDB Local seeder, uses static `local` credentials, loads `.env` itself, and refuses any endpoint whose host is not localhost/127.0.0.1/::1.
- `internal/routes`: `routes.go` sets global middleware (Recoverer, RequestID, request logger, CORS), `/health`, `/ready`, and mounts feature route groups in `registerRoutes`. Each feature gets its own file (e.g. `query.go`) that constructs service → handler and returns a `func(chi.Router)`.
- Layering: routes → `handler` → `service` → `repo`. Currently `QueryService`/`QueryHandler` are empty stubs and `/api/v1/query` has no endpoints.
- `internal/config`: `sync.Once` singleton from env vars; call `config.Init()` before `GetInstance()`. `Init` loads `.env` via godotenv (missing file ignored, real env vars win). `.env.sample` lists app keys only; `DYNAMODB_ENDPOINT` is seed-only.
- `internal/apperror` + `internal/constants`: typed `AppError` with code and HTTP status; error codes live in `constants/errors.go`.
- `internal/utils/response.go`: `JSON` / `JSONError` wrap payloads as `{"data": ...}` / `{"error": ...}`.
- Empty placeholder dirs (`model`, `repo`, `types`, `validations`, `middleware`, `routes/s2s`) mark where those concerns go. Pipeline stages go in `internal/service/<stage>/` (pure subpackages, wired by `QueryService`), external API wrappers in `internal/integrations/<name>/`. The full placement table is in the `go-conventions` skill; do not invent other packages.

## Project skills and hooks

- Skills in `server/.claude/skills/`: `go-conventions` (package layout, layering, errors, logging, tests, style; load before writing Go), `dynamodb-query-rules` (DynamoDB Query semantics, verified against AWS docs), `eval-harness` (eval case format, `make eval`, prompt-change workflow), `schema-authoring` (writing `schemas-local/*.json`).
- A Stop hook (`.claude/hooks/go-check.sh` at the repo root) runs `go vet ./...` and `go test ./...` in `server/` when Go files have uncommitted changes, and blocks once with the failure output. Fix failures; do not bypass it.
