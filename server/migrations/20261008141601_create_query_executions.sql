-- +goose Up
CREATE TABLE query_executions (
    id                 uuid PRIMARY KEY DEFAULT uuidv7(),
    query_id           uuid NOT NULL REFERENCES queries (id) ON DELETE CASCADE,
    start_key          jsonb,
    last_evaluated_key jsonb,
    item_count         integer NOT NULL,
    scanned_count      integer NOT NULL,
    consumed_capacity  numeric,
    error              text,
    duration_ms        integer NOT NULL,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX query_executions_query_id_idx ON query_executions (query_id);

CREATE TRIGGER query_executions_set_updated_at
    BEFORE UPDATE ON query_executions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE query_executions;
