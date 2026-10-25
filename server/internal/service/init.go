package service

import (
	"context"
	"time"

	"github.com/pgvector/pgvector-go"

	"github.com/akshaymundra/ai-text-to-nosql/internal/integrations/ollama"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
)

type tableSchemaStore interface {
	Save(ctx context.Context, schema model.TableSchema, patterns []model.AccessPattern) (model.TableSchema, error)
	Get(ctx context.Context, workspaceID, tableName string) (model.TableSchema, error)
	List(ctx context.Context, workspaceID string) ([]model.TableSchema, error)
	Delete(ctx context.Context, workspaceID, tableName string) error
}

type accessPatternSearcher interface {
	Search(
		ctx context.Context,
		workspaceID string,
		embedding pgvector.Vector,
		embeddingModel string,
		limit int,
	) ([]model.AccessPatternMatch, error)
}

type chatter interface {
	Chat(ctx context.Context, req ollama.ChatRequest) (string, error)
}

type embedder interface {
	Embed(ctx context.Context, model string, inputs []string) ([][]float32, error)
}

type workspaceStore interface {
	Create(ctx context.Context, ws model.Workspace) (model.Workspace, error)
	GetByName(ctx context.Context, name string) (model.Workspace, error)
}

type AuthService struct {
	workspaces           workspaceStore
	jwtSecret            []byte
	tokenTTL             time.Duration
	allowLocalConnection bool
}

func NewAuthService(
	workspaces workspaceStore,
	jwtSecret string,
	tokenTTL time.Duration,
	allowLocalConnection bool,
) *AuthService {
	return &AuthService{
		workspaces:           workspaces,
		jwtSecret:            []byte(jwtSecret),
		tokenTTL:             tokenTTL,
		allowLocalConnection: allowLocalConnection,
	}
}

type Retriever struct {
	patterns       accessPatternSearcher
	embedder       embedder
	embeddingModel string
}

func NewRetriever(patterns accessPatternSearcher, embedder embedder, embeddingModel string) *Retriever {
	return &Retriever{
		patterns:       patterns,
		embedder:       embedder,
		embeddingModel: embeddingModel,
	}
}

type Extractor struct {
	chat  chatter
	model string
	seed  int
}

func NewExtractor(chat chatter, model string, seed int) *Extractor {
	return &Extractor{chat: chat, model: model, seed: seed}
}

type WorkspaceService struct {
	allowLocalConnection bool
}

func NewWorkspaceService(allowLocalConnection bool) *WorkspaceService {
	return &WorkspaceService{allowLocalConnection: allowLocalConnection}
}

type TableSchemaService struct {
	schemas        tableSchemaStore
	embedder       embedder
	embeddingModel string
}

func NewTableSchemaService(schemas tableSchemaStore, embedder embedder, embeddingModel string) *TableSchemaService {
	return &TableSchemaService{
		schemas:        schemas,
		embedder:       embedder,
		embeddingModel: embeddingModel,
	}
}

type QueryService struct{}

func NewQueryService() *QueryService {
	return &QueryService{}
}
