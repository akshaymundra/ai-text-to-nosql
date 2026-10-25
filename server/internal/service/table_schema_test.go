package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pgvector/pgvector-go"

	"github.com/akshaymundra/ai-text-to-nosql/internal/apperror"
	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/repo"
	"github.com/akshaymundra/ai-text-to-nosql/internal/types"
)

const testEmbeddingModel = "qwen3-embedding:0.6b"

type fakeTableSchemaStore struct {
	savedSchema   model.TableSchema
	savedPatterns []model.AccessPattern
	getErr        error
	deleteErr     error
}

func (f *fakeTableSchemaStore) Save(
	ctx context.Context,
	schema model.TableSchema,
	patterns []model.AccessPattern,
) (model.TableSchema, error) {
	f.savedSchema = schema
	f.savedPatterns = patterns
	schema.ID = "ts-1"
	return schema, nil
}

func (f *fakeTableSchemaStore) Get(ctx context.Context, workspaceID, tableName string) (model.TableSchema, error) {
	return model.TableSchema{}, f.getErr
}

func (f *fakeTableSchemaStore) List(ctx context.Context, workspaceID string) ([]model.TableSchema, error) {
	return []model.TableSchema{}, nil
}

func (f *fakeTableSchemaStore) Delete(ctx context.Context, workspaceID, tableName string) error {
	return f.deleteErr
}

type fakeEmbedder struct {
	gotModel  string
	gotInputs []string
	err       error
}

func (f *fakeEmbedder) Embed(ctx context.Context, model string, inputs []string) ([][]float32, error) {
	f.gotModel = model
	f.gotInputs = inputs
	if f.err != nil {
		return nil, f.err
	}
	vectors := make([][]float32, len(inputs))
	for i := range inputs {
		vectors[i] = []float32{float32(i), 1}
	}
	return vectors, nil
}

func invoicesDefinition() dynamoschema.Table {
	return dynamoschema.Table{
		Name:         "invoices",
		Description:  "Invoices billed to organizations.",
		PartitionKey: "organization_id",
		SortKey:      "issued_at",
		Attributes: []dynamoschema.Attribute{
			{Name: "organization_id", Type: dynamoschema.TypeString},
			{Name: "issued_at", Type: dynamoschema.TypeString, Format: dynamoschema.FormatTimestamp},
			{Name: "status", Type: dynamoschema.TypeString},
		},
		Indexes: []dynamoschema.Index{{Name: "status-issued-at-index", PartitionKey: "status", SortKey: "issued_at"}},
		AccessPatterns: []dynamoschema.AccessPattern{
			{Description: "List invoices of an organization."},
			{Description: "List invoices with a given payment status.", Index: "status-issued-at-index"},
		},
	}
}

func TestPutTableSchema(t *testing.T) {
	store := &fakeTableSchemaStore{}
	embedder := &fakeEmbedder{}
	svc := NewTableSchemaService(store, embedder, testEmbeddingModel)
	definition := invoicesDefinition()

	saved, err := svc.Put(context.Background(), "ws-1", "invoices", types.PutTableSchemaRequest{Definition: definition})
	if err != nil {
		t.Fatalf("put: %v", err)
	}

	wantInputs := []string{
		"Table invoices: Invoices billed to organizations.\nAccess pattern: List invoices of an organization.",
		"Table invoices: Invoices billed to organizations.\nAccess pattern: List invoices with a given payment status.",
	}
	if diff := cmp.Diff(wantInputs, embedder.gotInputs); diff != "" {
		t.Errorf("embedded texts mismatch (-want +got):\n%s", diff)
	}
	if embedder.gotModel != testEmbeddingModel {
		t.Errorf("embedding model = %q, want %q", embedder.gotModel, testEmbeddingModel)
	}

	wantHash, err := definition.ContentHash()
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	wantSchema := model.TableSchema{
		WorkspaceID: "ws-1",
		TableName:   "invoices",
		Definition:  definition,
		ContentHash: wantHash,
	}
	if diff := cmp.Diff(wantSchema, store.savedSchema); diff != "" {
		t.Errorf("saved schema mismatch (-want +got):\n%s", diff)
	}
	if saved.ID != "ts-1" {
		t.Errorf("returned schema ID = %q, want the stored one", saved.ID)
	}

	statusIndex := "status-issued-at-index"
	wantPatterns := []model.AccessPattern{
		{
			Description:    "List invoices of an organization.",
			Embedding:      pgvector.NewVector([]float32{0, 1}),
			EmbeddingModel: testEmbeddingModel,
		},
		{
			IndexName:      &statusIndex,
			Description:    "List invoices with a given payment status.",
			Embedding:      pgvector.NewVector([]float32{1, 1}),
			EmbeddingModel: testEmbeddingModel,
		},
	}
	if diff := cmp.Diff(wantPatterns, store.savedPatterns, cmp.AllowUnexported(pgvector.Vector{})); diff != "" {
		t.Errorf("saved patterns mismatch (-want +got):\n%s", diff)
	}
}

func TestPutTableSchemaRejects(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		mutate    func(d *dynamoschema.Table)
		wantMsg   string
	}{
		{
			name:      "url table name differs from definition",
			tableName: "bills",
			mutate:    func(d *dynamoschema.Table) {},
			wantMsg:   `definition table "invoices" does not match "bills" in the URL`,
		},
		{
			name:      "invalid definition",
			tableName: "invoices",
			mutate:    func(d *dynamoschema.Table) { d.PartitionKey = "org_id"; d.AccessPatterns = nil },
			wantMsg:   `key "org_id" of table is not in attributes; at least one access pattern is required`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeTableSchemaStore{}
			embedder := &fakeEmbedder{}
			svc := NewTableSchemaService(store, embedder, testEmbeddingModel)
			definition := invoicesDefinition()
			tc.mutate(&definition)

			_, err := svc.Put(context.Background(), "ws-1", tc.tableName, types.PutTableSchemaRequest{Definition: definition})

			assertAppErrorCode(t, err, constants.ErrCodeBadRequest)
			var appErr *apperror.AppError
			if errors.As(err, &appErr) && !strings.Contains(appErr.Message, tc.wantMsg) {
				t.Errorf("message = %q, want it to contain %q", appErr.Message, tc.wantMsg)
			}
			if embedder.gotInputs != nil {
				t.Error("embedder was called for a rejected schema")
			}
		})
	}
}

func TestPutTableSchemaEmbedFailure(t *testing.T) {
	embedErr := errors.New("connection refused")
	store := &fakeTableSchemaStore{}
	svc := NewTableSchemaService(store, &fakeEmbedder{err: embedErr}, testEmbeddingModel)

	_, err := svc.Put(
		context.Background(),
		"ws-1",
		"invoices",
		types.PutTableSchemaRequest{Definition: invoicesDefinition()},
	)

	if !errors.Is(err, embedErr) {
		t.Fatalf("err = %v, want it to wrap %v", err, embedErr)
	}
	if store.savedSchema.TableName != "" {
		t.Error("schema was saved although embedding failed")
	}
}

func TestTableSchemaNotFound(t *testing.T) {
	store := &fakeTableSchemaStore{getErr: repo.ErrNotFound, deleteErr: repo.ErrNotFound}
	svc := NewTableSchemaService(store, &fakeEmbedder{}, testEmbeddingModel)

	_, err := svc.Get(context.Background(), "ws-1", "invoices")
	assertAppErrorCode(t, err, constants.ErrCodeNotFound)

	err = svc.Delete(context.Background(), "ws-1", "invoices")
	assertAppErrorCode(t, err, constants.ErrCodeNotFound)
}
