package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/akshaymundra/ai-text-to-nosql/evals"
	"github.com/akshaymundra/ai-text-to-nosql/internal/config"
	"github.com/akshaymundra/ai-text-to-nosql/internal/integrations/ollama"
	"github.com/akshaymundra/ai-text-to-nosql/internal/postgres"
	"github.com/akshaymundra/ai-text-to-nosql/internal/repo"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/extraction"
)

func main() {
	dir := flag.String("dir", "evals/cases", "case directory: evals/cases (tuning) or evals/holdout (never tune against it)")
	flag.Parse()

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
	cases, err := evals.LoadCases(*dir)
	if err != nil {
		log.Fatal(err)
	}

	client := ollama.NewClient(cfg.OllamaURL, cfg.OllamaTimeout)
	runner := service.NewQueryRunner(
		service.NewRetriever(repo.NewAccessPatternRepo(db), client, cfg.EmbeddingModel),
		service.NewExtractor(client, cfg.ExtractionModel, cfg.ExtractionSeed),
		service.NewFieldSuggester(client, cfg.EmbeddingModel),
	)

	fmt.Printf("model %s  seed %d  prompt %s  embeddings %s  cases %d from %s\n\n",
		cfg.ExtractionModel, cfg.ExtractionSeed, extraction.PromptVersion, cfg.EmbeddingModel, len(cases), *dir)

	start := time.Now()
	results := make([]evals.CaseResult, 0, len(cases))
	for _, c := range cases {
		caseStart := time.Now()
		run, err := runner.Run(ctx, workspace.ID, c.Request, c.Now)
		if err != nil {
			log.Fatalf("%s: %v", c.ID, err)
		}
		result, err := evals.Evaluate(c, run)
		if err != nil {
			log.Fatalf("%s: %v", c.ID, err)
		}
		results = append(results, result)
		printResult(result, time.Since(caseStart))
	}

	printSummary(results, time.Since(start))
}

func printResult(r evals.CaseResult, took time.Duration) {
	label := "PASS"
	if !r.Passed {
		label = "FAIL"
	}
	fmt.Printf("%s %-40s %5.1fs  %s\n", label, r.Case.ID, took.Seconds(), r.Case.Request)
	for _, rejection := range r.Run.Result.Rejections {
		if rejection.Suggestion != "" {
			fmt.Printf("     suggestion: %q -> %q\n", rejection.Field, rejection.Suggestion)
		}
	}
	if r.Passed {
		return
	}
	fmt.Printf("     expect %s\n     got    %s\n", compact(r.Case.Expect), compact(r.Got))
	if r.IRDiff != "" {
		fmt.Printf("     ir ref %s\n     ir got %s\n",
			compact(evals.Extraction{IR: r.Case.IR, UnmappedTerms: r.Case.UnmappedTerms}),
			compact(evals.Extraction{IR: r.Run.Extraction.Query, UnmappedTerms: r.Run.Extraction.UnmappedTerms}),
		)
	} else {
		fmt.Println("     ir matches the reference: pipeline bug, add a unit test")
	}
}

func printSummary(results []evals.CaseResult, took time.Duration) {
	passed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		}
	}
	fmt.Println("\nby tag:")
	for _, s := range evals.ScoreByTag(results) {
		fmt.Printf("  %-16s %2d/%-2d\n", s.Tag, s.Passed, s.Total)
	}
	fmt.Printf("\nscore %d/%d (%.0f%%) in %s\n",
		passed, len(results), 100*float64(passed)/float64(len(results)), took.Round(time.Second))
}

func compact(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(data)
}
