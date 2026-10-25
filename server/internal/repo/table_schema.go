package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
)

type TableSchemaRepo struct {
	db *pgxpool.Pool
}

func NewTableSchemaRepo(db *pgxpool.Pool) *TableSchemaRepo {
	return &TableSchemaRepo{db: db}
}

// Save upserts a table schema and replaces all of its access patterns in one transaction, so retrieval never
// sees a schema with patterns from an older version.
func (r *TableSchemaRepo) Save(
	ctx context.Context,
	schema model.TableSchema,
	patterns []model.AccessPattern,
) (model.TableSchema, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return model.TableSchema{}, fmt.Errorf("saving table schema %q: %w", schema.TableName, err)
	}
	defer tx.Rollback(ctx)

	saved, err := upsertTableSchema(ctx, tx, schema)
	if err != nil {
		return model.TableSchema{}, err
	}
	if err := replaceAccessPatterns(ctx, tx, saved, patterns); err != nil {
		return model.TableSchema{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.TableSchema{}, fmt.Errorf("saving table schema %q: %w", schema.TableName, err)
	}
	return saved, nil
}

func (r *TableSchemaRepo) Get(ctx context.Context, workspaceID, tableName string) (model.TableSchema, error) {
	rows, err := r.db.Query(ctx,
		`SELECT * FROM table_schemas WHERE workspace_id = $1 AND table_name = $2`,
		workspaceID, tableName,
	)
	if err != nil {
		return model.TableSchema{}, fmt.Errorf("getting table schema %q: %w", tableName, err)
	}

	schema, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.TableSchema])
	if errors.Is(err, pgx.ErrNoRows) {
		return model.TableSchema{}, ErrNotFound
	}
	if err != nil {
		return model.TableSchema{}, fmt.Errorf("getting table schema %q: %w", tableName, err)
	}
	return schema, nil
}

func (r *TableSchemaRepo) List(ctx context.Context, workspaceID string) ([]model.TableSchema, error) {
	rows, err := r.db.Query(ctx,
		`SELECT * FROM table_schemas WHERE workspace_id = $1 ORDER BY table_name`,
		workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing table schemas: %w", err)
	}

	schemas, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.TableSchema])
	if err != nil {
		return nil, fmt.Errorf("listing table schemas: %w", err)
	}
	return schemas, nil
}

func (r *TableSchemaRepo) Delete(ctx context.Context, workspaceID, tableName string) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM table_schemas WHERE workspace_id = $1 AND table_name = $2`,
		workspaceID, tableName,
	)
	if err != nil {
		return fmt.Errorf("deleting table schema %q: %w", tableName, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func upsertTableSchema(ctx context.Context, tx pgx.Tx, schema model.TableSchema) (model.TableSchema, error) {
	rows, err := tx.Query(ctx,
		`INSERT INTO table_schemas (workspace_id, table_name, definition, content_hash)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (workspace_id, table_name)
		DO UPDATE SET definition = EXCLUDED.definition, content_hash = EXCLUDED.content_hash
		RETURNING *`,
		schema.WorkspaceID, schema.TableName, schema.Definition, schema.ContentHash,
	)
	if err != nil {
		return model.TableSchema{}, fmt.Errorf("upserting table schema %q: %w", schema.TableName, err)
	}

	saved, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.TableSchema])
	if err != nil {
		return model.TableSchema{}, fmt.Errorf("upserting table schema %q: %w", schema.TableName, err)
	}
	return saved, nil
}

func replaceAccessPatterns(
	ctx context.Context,
	tx pgx.Tx,
	schema model.TableSchema,
	patterns []model.AccessPattern,
) error {
	if _, err := tx.Exec(ctx, `DELETE FROM access_patterns WHERE table_schema_id = $1`, schema.ID); err != nil {
		return fmt.Errorf("deleting access patterns of %q: %w", schema.TableName, err)
	}

	for _, p := range patterns {
		_, err := tx.Exec(ctx,
			`INSERT INTO access_patterns
				(workspace_id, table_schema_id, index_name, description, embedding, embedding_model)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			schema.WorkspaceID, schema.ID, p.IndexName, p.Description, p.Embedding, p.EmbeddingModel,
		)
		if err != nil {
			return fmt.Errorf("inserting access pattern of %q: %w", schema.TableName, err)
		}
	}
	return nil
}
