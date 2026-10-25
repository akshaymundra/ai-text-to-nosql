package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/akshaymundra/ai-text-to-nosql/evals"
	"github.com/akshaymundra/ai-text-to-nosql/internal/config"
	"github.com/akshaymundra/ai-text-to-nosql/internal/integrations/ollama"
	"github.com/akshaymundra/ai-text-to-nosql/internal/postgres"
	"github.com/akshaymundra/ai-text-to-nosql/internal/repo"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service"
)

func main() {
	config.Init()
	cfg := config.GetInstance()
	ctx := context.Background()

	db, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.DBPingTimeout)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	workspace, err := repo.NewWorkspaceRepo(db).GetByName(ctx, cfg.DemoWorkspace)
	if err != nil {
		log.Fatalf("workspace %q: %v (run make register-demo first)", cfg.DemoWorkspace, err)
	}
	cases, err := evals.LoadCases("evals/cases")
	if err != nil {
		log.Fatal(err)
	}

	retriever := service.NewRetriever(
		repo.NewAccessPatternRepo(db),
		ollama.NewClient(cfg.OllamaURL, cfg.OllamaTimeout),
		cfg.EmbeddingModel,
	)

	firstHits, anyHits := 0, 0
	for _, c := range cases {
		retrieved, err := retriever.Retrieve(ctx, workspace.ID, c.Request)
		if err != nil {
			log.Fatalf("%s: %v", c.ID, err)
		}
		rank := evals.RetrievalRank(c.IR.Entity, retrieved.Tables)
		if rank == 1 {
			firstHits++
		}
		if rank > 0 {
			anyHits++
		}
		fmt.Printf("%-5s %-40s want %-14s got %s\n", label(rank), c.ID, c.IR.Entity, tableNames(retrieved))
	}

	fmt.Printf("\nrecall@1 %d/%d (%.0f%%)  recall@%d %d/%d (%.0f%%)\n",
		firstHits, len(cases), percent(firstHits, len(cases)),
		service.RetrievalMaxTables, anyHits, len(cases), percent(anyHits, len(cases)),
	)
}

func label(rank int) string {
	switch rank {
	case 0:
		return "MISS"
	case 1:
		return "HIT"
	default:
		return fmt.Sprintf("HIT%d", rank)
	}
}

func tableNames(r service.Retrieved) string {
	names := make([]string, len(r.Tables))
	for i, t := range r.Tables {
		names[i] = t.Name
	}
	return "[" + strings.Join(names, " ") + "]"
}

func percent(n, total int) float64 {
	return 100 * float64(n) / float64(total)
}
