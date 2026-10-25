package evals

import (
	"slices"
	"strings"

	"github.com/google/go-cmp/cmp"

	"github.com/akshaymundra/ai-text-to-nosql/internal/service"
)

type CaseResult struct {
	Case   Case
	Run    service.Run
	Got    Outcome
	Passed bool
	Diff   string
	IRDiff string
}

type TagScore struct {
	Tag    string
	Passed int
	Total  int
}

// Evaluate scores one run. Pass or fail compares outcomes only; IRDiff says whether a failure started in
// extraction (the model's IR differs from the reference) or later in the pipeline.
func Evaluate(c Case, run service.Run) (CaseResult, error) {
	got, err := OutcomeOf(run.Result)
	if err != nil {
		return CaseResult{}, err
	}
	diff := cmp.Diff(c.Expect, got)

	reference := Extraction{IR: c.IR, UnmappedTerms: c.UnmappedTerms}
	extracted := Extraction{IR: run.Extraction.Query, UnmappedTerms: run.Extraction.UnmappedTerms}

	return CaseResult{
		Case:   c,
		Run:    run,
		Got:    got,
		Passed: diff == "",
		Diff:   diff,
		IRDiff: cmp.Diff(reference, extracted, cmp.Comparer(sameTerms)),
	}, nil
}

// ScoreByTag counts passes per tag, sorted by tag name.
func ScoreByTag(results []CaseResult) []TagScore {
	byTag := map[string]*TagScore{}
	for _, r := range results {
		for _, tag := range r.Case.Tags {
			score, ok := byTag[tag]
			if !ok {
				score = &TagScore{Tag: tag}
				byTag[tag] = score
			}
			score.Total++
			if r.Passed {
				score.Passed++
			}
		}
	}

	scores := make([]TagScore, 0, len(byTag))
	for _, s := range byTag {
		scores = append(scores, *s)
	}
	slices.SortFunc(scores, func(a, b TagScore) int { return strings.Compare(a.Tag, b.Tag) })
	return scores
}

// sameTerms treats a missing list and an empty one as equal: the model returns [] where cases omit the key.
func sameTerms(a, b []string) bool {
	return len(a) == 0 && len(b) == 0 || slices.Equal(a, b)
}
