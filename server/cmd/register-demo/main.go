package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/akshaymundra/ai-text-to-nosql/internal/config"
	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/integrations/httpclient"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/types"
)

const requestTimeout = 2 * time.Minute

type envelope[T any] struct {
	Data T `json:"data"`
}

func main() {
	config.Init()
	cfg := config.GetInstance()
	if cfg.DemoWorkspacePassword == "" {
		log.Fatal("DEMO_WORKSPACE_PASSWORD is required")
	}

	tables, err := dynamoschema.LoadDir("schemas-local")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	// 127.0.0.1, not localhost: another dev server listening on [::1] with the same port would answer instead.
	api := httpclient.New("http://127.0.0.1:"+cfg.ServerPort+"/api/v1", requestTimeout)
	token, err := registerOrLogin(ctx, api, cfg)
	if err != nil {
		log.Fatal(err)
	}

	authed := api.WithHeader("Authorization", "Bearer "+token)
	for _, table := range tables {
		req := types.PutTableSchemaRequest{Definition: table}
		if err := authed.Put(ctx, "/schemas/"+table.Name, req, nil); err != nil {
			log.Fatalf("%s: %v", table.Name, err)
		}
		log.Printf("%s: registered, %d access patterns embedded", table.Name, len(table.AccessPatterns))
	}
}

func registerOrLogin(ctx context.Context, api *httpclient.Client, cfg *config.Config) (string, error) {
	var resp envelope[types.TokenResponse]
	err := api.Post(ctx, "/auth/register", types.RegisterRequest{
		Name:           cfg.DemoWorkspace,
		Password:       cfg.DemoWorkspacePassword,
		ConnectionType: model.ConnectionTypeLocal,
		AWSRegion:      cfg.DynamoDBRegion,
		EndpointURL:    &cfg.DynamoDBEndpoint,
	}, &resp)

	var statusErr *httpclient.StatusError
	if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusConflict {
		log.Printf("workspace %q exists, logging in", cfg.DemoWorkspace)
		login := types.LoginRequest{Name: cfg.DemoWorkspace, Password: cfg.DemoWorkspacePassword}
		err = api.Post(ctx, "/auth/login", login, &resp)
	}
	if err != nil {
		return "", err
	}
	return resp.Data.Token, nil
}
