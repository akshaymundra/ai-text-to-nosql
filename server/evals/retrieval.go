package evals

import "github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"

// RetrievalRank is the 1-based position of entity among the retrieved tables, or 0 when it was not retrieved.
func RetrievalRank(entity string, tables []dynamoschema.Table) int {
	for i, t := range tables {
		if t.Name == entity {
			return i + 1
		}
	}
	return 0
}
