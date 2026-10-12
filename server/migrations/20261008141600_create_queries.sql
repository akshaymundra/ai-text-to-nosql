-- +goose Up
CREATE TABLE queries (
    id              uuid PRIMARY KEY DEFAULT uuidv7(),
    workspace_id    uuid NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    table_schema_id uuid REFERENCES table_schemas (id) ON DELETE SET NULL,
    user_text       text NOT NULL,
    retrieved       jsonb NOT NULL DEFAULT '[]',
    ir              jsonb,
    status          text NOT NULL CHECK (status IN ('valid', 'rejected')),
    rejection_code  text,
    rejection_field text,
    table_name      text,
    index_name      text,
    schema_hash     text,
    query_input     jsonb,
    warnings        jsonb NOT NULL DEFAULT '[]',
    explanation     text NOT NULL,
    llm_model       text NOT NULL,
    prompt_version  text NOT NULL,
    latency_ms      integer NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT queries_status_fields CHECK (
        (status = 'valid' AND query_input IS NOT NULL AND rejection_code IS NULL)
        OR (status = 'rejected' AND rejection_code IS NOT NULL)
    )
);

CREATE INDEX queries_workspace_id_created_at_idx ON queries (workspace_id, created_at DESC);
CREATE INDEX queries_table_schema_id_idx ON queries (table_schema_id);

CREATE TRIGGER queries_set_updated_at
    BEFORE UPDATE ON queries
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE queries;
