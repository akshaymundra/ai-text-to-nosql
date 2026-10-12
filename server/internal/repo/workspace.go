package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
)

type WorkspaceRepo struct {
	db *pgxpool.Pool
}

func NewWorkspaceRepo(db *pgxpool.Pool) *WorkspaceRepo {
	return &WorkspaceRepo{db: db}
}

func (r *WorkspaceRepo) Create(ctx context.Context, ws model.Workspace) (model.Workspace, error) {
	rows, err := r.db.Query(ctx,
		`INSERT INTO workspaces (name, password_hash, connection_type, aws_region, endpoint_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING *`,
		ws.Name, ws.PasswordHash, ws.ConnectionType, ws.AWSRegion, ws.EndpointURL,
	)
	if err != nil {
		return model.Workspace{}, fmt.Errorf("inserting workspace %q: %w", ws.Name, err)
	}

	created, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Workspace])
	if isUniqueViolation(err) {
		return model.Workspace{}, ErrAlreadyExists
	}
	if err != nil {
		return model.Workspace{}, fmt.Errorf("inserting workspace %q: %w", ws.Name, err)
	}
	return created, nil
}

func (r *WorkspaceRepo) GetByName(ctx context.Context, name string) (model.Workspace, error) {
	rows, err := r.db.Query(ctx, `SELECT * FROM workspaces WHERE name = $1`, name)
	if err != nil {
		return model.Workspace{}, fmt.Errorf("getting workspace %q: %w", name, err)
	}

	ws, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Workspace])
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Workspace{}, ErrNotFound
	}
	if err != nil {
		return model.Workspace{}, fmt.Errorf("getting workspace %q: %w", name, err)
	}
	return ws, nil
}
