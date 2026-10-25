package evals

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/compiler"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
)

type Case struct {
	ID      string    `json:"id"`
	Request string    `json:"request"`
	Now     time.Time `json:"now"`
	Tags    []string  `json:"tags"`
	IR      ir.Query  `json:"ir"`
	// UnmappedTerms are request words that name no field; a correct extraction reports them instead of guessing.
	UnmappedTerms []string `json:"unmapped_terms,omitempty"`
	Expect        Outcome  `json:"expect"`
}

type Outcome struct {
	Status model.QueryStatus       `json:"status"`
	Code   constants.RejectionCode `json:"code,omitempty"`
	Field  string                  `json:"field,omitempty"`
	Table  string                  `json:"table,omitempty"`
	Index  string                  `json:"index,omitempty"`
	Query  *Query                  `json:"query,omitempty"`
}

type Query struct {
	KeyConditionExpression    string            `json:"KeyConditionExpression"`
	FilterExpression          string            `json:"FilterExpression,omitempty"`
	ExpressionAttributeNames  map[string]string `json:"ExpressionAttributeNames"`
	ExpressionAttributeValues map[string]any    `json:"ExpressionAttributeValues"`
}

func LoadCases(dir string) ([]Case, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}

	cases := make([]Case, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var c Case
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&c); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		cases = append(cases, c)
	}
	return cases, nil
}

// OutcomeOf turns a pipeline result into the shape cases are written in: the first rejection only,
// and attribute values as plain JSON values instead of DynamoDB's typed form.
func OutcomeOf(result service.QueryResult) (Outcome, error) {
	if result.Status == model.QueryStatusRejected {
		first := result.Rejections[0]
		return Outcome{Status: result.Status, Code: first.Code, Field: first.Field}, nil
	}

	values := make(map[string]any, len(result.Query.ExpressionAttributeValues))
	for placeholder, av := range result.Query.ExpressionAttributeValues {
		v, err := plainValue(av)
		if err != nil {
			return Outcome{}, fmt.Errorf("value %s: %w", placeholder, err)
		}
		values[placeholder] = v
	}

	return Outcome{
		Status: result.Status,
		Table:  result.Table,
		Index:  result.Index,
		Query: &Query{
			KeyConditionExpression:    result.Query.KeyConditionExpression,
			FilterExpression:          result.Query.FilterExpression,
			ExpressionAttributeNames:  result.Query.ExpressionAttributeNames,
			ExpressionAttributeValues: values,
		},
	}, nil
}

func plainValue(av compiler.AttributeValue) (any, error) {
	if s, ok := av["S"]; ok {
		return s, nil
	}
	if n, ok := av["N"]; ok {
		return strconv.ParseFloat(n, 64)
	}
	return nil, fmt.Errorf("unsupported attribute value %v", av)
}
