-- +goose Up
CREATE TABLE access_patterns (
    id              uuid PRIMARY KEY DEFAULT uuidv7(),
    workspace_id    uuid NOT NULL,
    table_schema_id uuid NOT NULL,
    index_name      text,
    description     text NOT NULL,
    embedding       vector(1024) NOT NULL,
    embedding_model text NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    -- workspace_id is denormalized for filtered retrieval; the composite key stops it drifting from its schema's workspace.
    FOREIGN KEY (table_schema_id, workspace_id)
        REFERENCES table_schemas (id, workspace_id) ON DELETE CASCADE
);

CREATE INDEX access_patterns_embedding_idx ON access_patterns USING hnsw (embedding vector_cosine_ops);
CREATE INDEX access_patterns_workspace_id_idx ON access_patterns (workspace_id);
CREATE INDEX access_patterns_table_schema_id_idx ON access_patterns (table_schema_id);

CREATE TRIGGER access_patterns_set_updated_at
    BEFORE UPDATE ON access_patterns
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE access_patterns;
