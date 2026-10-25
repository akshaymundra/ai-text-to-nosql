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
