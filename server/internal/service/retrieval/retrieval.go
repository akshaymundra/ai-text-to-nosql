package retrieval

import "github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"

// qwen3-embedding expects an instruction on queries only; stored documents are embedded as plain text.
// The format, including no space after "Query:", follows the model card.
const queryInstruction = "Instruct: Given a request for data, retrieve the database table access patterns that can answer it\nQuery:"

func DocumentText(table dynamoschema.Table, pattern dynamoschema.AccessPattern) string {
	return "Table " + table.Name + ": " + table.Description + "\nAccess pattern: " + pattern.Description
}

func QueryText(request string) string {
	return queryInstruction + request
}

// SelectTables keeps each table once, at the rank of its best matching access pattern, up to max tables.
func SelectTables(ranked []dynamoschema.Table, max int) []dynamoschema.Table {
	selected := []dynamoschema.Table{}
	seen := map[string]bool{}
	for _, table := range ranked {
		if len(selected) == max {
			break
		}
		if seen[table.Name] {
			continue
		}
		seen[table.Name] = true
		selected = append(selected, table)
	}
	return selected
}
