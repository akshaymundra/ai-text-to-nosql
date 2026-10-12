# Postgres ERD

App data only. DynamoDB Local is the demo customer database, never app storage. Tables are created by goose migrations in `server/migrations/`.

```mermaid
erDiagram
    WORKSPACES ||--o{ TABLE_SCHEMAS : "registers"
    WORKSPACES ||--o{ ACCESS_PATTERNS : "scopes retrieval"
    TABLE_SCHEMAS ||--o{ ACCESS_PATTERNS : "described by"
    WORKSPACES ||--o{ QUERIES : "receives"
    TABLE_SCHEMAS |o--o{ QUERIES : "compiled against"
    QUERIES ||--o{ QUERY_EXECUTIONS : "executed as"

    WORKSPACES {
        uuid id PK "uuidv7()"
        text name UK "login name, stored lowercase"
        text password_hash "bcrypt; raw password never stored"
        text connection_type "CHECK temporary_credentials | local"
        text aws_region
        text endpoint_url "nullable; required when local"
        bytea aws_credentials_enc "nullable; AES-256-GCM of access key id, secret, session token"
        timestamptz aws_credentials_expires_at "nullable; set together with aws_credentials_enc"
        timestamptz created_at
        timestamptz updated_at
    }

    TABLE_SCHEMAS {
        uuid id PK "uuidv7()"
        uuid workspace_id FK "ON DELETE CASCADE; UK with table_name"
        text table_name "UK with workspace_id"
        jsonb definition "full schema.Table"
        text content_hash "sha256 of definition"
        timestamptz created_at
        timestamptz updated_at
    }

    ACCESS_PATTERNS {
        uuid id PK "uuidv7()"
        uuid workspace_id FK "composite FK with table_schema_id"
        uuid table_schema_id FK "ON DELETE CASCADE"
        text index_name "nullable; null = base table PK/SK"
        text description "text that was embedded"
        vector embedding "vector(1024), HNSW cosine"
        text embedding_model "e.g. qwen3-embedding:0.6b"
        timestamptz created_at
        timestamptz updated_at
    }

    QUERIES {
        uuid id PK "uuidv7(); query_id used by /execute"
        uuid workspace_id FK "ON DELETE CASCADE"
        uuid table_schema_id FK "nullable; ON DELETE SET NULL"
        text user_text "natural-language request"
        jsonb retrieved "access pattern ids + distances"
        jsonb ir "nullable; LLM output, null if parse failed"
        text status "CHECK valid | rejected"
        text rejection_code "nullable; required when rejected"
        text rejection_field "nullable"
        text table_name "nullable; kept if schema deleted"
        text index_name "nullable; planner output"
        text schema_hash "nullable; content_hash at compile time"
        jsonb query_input "nullable; required when valid"
        jsonb warnings "array, empty if none"
        text explanation
        text llm_model "pinned Ollama tag"
        text prompt_version
        int latency_ms
        timestamptz created_at
        timestamptz updated_at
    }

    QUERY_EXECUTIONS {
        uuid id PK "uuidv7()"
        uuid query_id FK "ON DELETE CASCADE"
        jsonb start_key "nullable; ExclusiveStartKey in"
        jsonb last_evaluated_key "nullable; cursor out, null = last page"
        int item_count
        int scanned_count
        numeric consumed_capacity "nullable; RCUs"
        text error "nullable"
        int duration_ms
        timestamptz created_at
        timestamptz updated_at
    }
```

## Notes

- **Deletes are hard.** No `deleted_at`. Child rows go with `ON DELETE CASCADE`; `queries.table_schema_id` is `SET NULL` so history survives a schema delete.
- **`updated_at`** is maintained by the shared `set_updated_at()` trigger on every table, not by app code.
- **Login:** workspace `name` + password, checked against `password_hash`. Sessions are JWTs (not stored).
- **AWS credentials (Phase 1):** the user pastes temporary credentials (`ASIA...` key with a session token; long-lived `AKIA...` keys are rejected). They are encrypted with AES-256-GCM using `CREDENTIALS_ENCRYPTION_KEY`, with `workspace_id` as additional authenticated data, and decrypted only in memory per execute. Expired credentials return `credentials_expired`. AssumeRole (`role_arn` + `external_id`) replaces this later as a new `connection_type`.
- **`access_patterns.workspace_id`** is denormalized so retrieval filters by workspace without a join. The composite FK to `table_schemas (id, workspace_id)` guarantees it always matches the schema's workspace.
- **Execute guard:** before running a saved query, `queries.schema_hash` must equal the current `table_schemas.content_hash`; otherwise the schema changed and the query is rejected.
- **No query results are stored.** `query_executions` keeps counts and cursors only.
