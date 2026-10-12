-- +goose Up
CREATE TABLE table_schemas (
    id           uuid PRIMARY KEY DEFAULT uuidv7(),
    workspace_id uuid NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    table_name   text NOT NULL,
    definition   jsonb NOT NULL,
    content_hash text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, table_name),
    -- Target of the composite foreign key in access_patterns; not redundant with the primary key.
    UNIQUE (id, workspace_id)
);

CREATE TRIGGER table_schemas_set_updated_at
    BEFORE UPDATE ON table_schemas
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE table_schemas;
