-- +goose Up
CREATE TABLE workspaces (
    id                         uuid PRIMARY KEY DEFAULT uuidv7(),
    name                       text NOT NULL UNIQUE CHECK (name = lower(name)),
    password_hash              text NOT NULL,
    connection_type            text NOT NULL CHECK (connection_type IN ('temporary_credentials', 'local')),
    aws_region                 text NOT NULL,
    endpoint_url               text,
    aws_credentials_enc        bytea,
    aws_credentials_expires_at timestamptz,
    created_at                 timestamptz NOT NULL DEFAULT now(),
    updated_at                 timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT workspaces_local_requires_endpoint
        CHECK (connection_type <> 'local' OR endpoint_url IS NOT NULL),
    CONSTRAINT workspaces_credentials_have_expiry
        CHECK ((aws_credentials_enc IS NULL) = (aws_credentials_expires_at IS NULL))
);

CREATE TRIGGER workspaces_set_updated_at
    BEFORE UPDATE ON workspaces
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE workspaces;
