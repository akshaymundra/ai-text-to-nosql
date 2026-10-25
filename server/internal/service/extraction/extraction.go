package extraction

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
)

// PromptVersion is stored with every query so eval scores and saved queries can be traced to the prompt that
// produced them. Bump it whenever the prompt or schema below changes.
const PromptVersion = "extraction-v2"

const rules = `You turn a request for data into a JSON description of the filters it asks for. You never write database queries.

Output:
- entity: the one table the request is about.
- conditions: one entry per filter the request states. Use only fields of the chosen table.
- unmapped_terms: words of the request that name a field no table has, for example "phone number" when no table has a phone field. Leave it empty when every filter maps to a field.

Operators:
- eq, ne, lt, lte, gt, gte: compare the field with one value.
- between: an inclusive range; value is [low, high].
- in: value is the list of allowed values.
- begins_with: value is a text prefix.
- contains: value is text the field contains.
- exists, not_exists: the field is present or missing; no value.

Rules:
- Every identifier or name the request mentions (an organization, a user, a project, an ID like x_123) becomes a condition. Never drop one.
- A value shaped like a field's ID examples (a lowercase slug, a prefixed code) filters that ID field, not a name or title field.
- Copy values exactly as the request or the field description spells them. Never translate or invent values.
- Numbers are JSON numbers; text is a JSON string.
- Dates: write whole days as YYYY-MM-DD and never add a time or time zone to them; the system does that. Use YYYY-MM-DDTHH:MM:SSZ only when the request names a time of day.
- A named period (a day, a month, a quarter, a year, last week, the previous month) covers the whole calendar period: use between with its first and last day. The previous month means the previous calendar month, not the last 30 days.
- Use gte, gt, lte or lt only for open ranges ("since", "after", "before", "until").
- Resolve relative dates against today's date.
- A range on one field is one between condition, never two conditions.
- exists and not_exists take no value: leave value out.
- Add no condition the request does not state. Sorting and result limits are not part of the output.`

type Extraction struct {
	Query         ir.Query
	UnmappedTerms []string
}

type answer struct {
	Entity        string         `json:"entity"`
	Conditions    []ir.Condition `json:"conditions"`
	UnmappedTerms []string       `json:"unmapped_terms"`
}

// Schema constrains the model's answer: entity and field are limited to the retrieved tables, so a field name
// outside them cannot be produced at all.
func Schema(tables []dynamoschema.Table) (json.RawMessage, error) {
	scalar := []any{map[string]any{"type": "string"}, map[string]any{"type": "number"}}
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"entity": map[string]any{"type": "string", "enum": tableNames(tables)},
			"conditions": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"field":    map[string]any{"type": "string", "enum": fieldNames(tables)},
						"operator": map[string]any{"type": "string", "enum": ir.Operators()},
						"value": map[string]any{
							"anyOf": append(scalar, map[string]any{"type": "array", "items": map[string]any{"anyOf": scalar}}),
						},
					},
					"required": []string{"field", "operator"},
				},
			},
			"unmapped_terms": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required": []string{"entity", "conditions", "unmapped_terms"},
	}
	return json.Marshal(schema)
}

func SystemPrompt(tables []dynamoschema.Table, now time.Time) string {
	var b strings.Builder
	b.WriteString(rules)
	fmt.Fprintf(&b, "\n\nToday is %s (UTC).\n\nTables:\n", now.UTC().Format("Monday, 2006-01-02"))
	for _, t := range tables {
		fmt.Fprintf(&b, "\n## %s\n%s\nFields:\n", t.Name, t.Description)
		for _, a := range t.Attributes {
			fmt.Fprintf(&b, "- %s (%s): %s\n", a.Name, attributeKind(a), a.Description)
		}
	}
	return b.String()
}

func Parse(content string) (Extraction, error) {
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.DisallowUnknownFields()

	var a answer
	if err := decoder.Decode(&a); err != nil {
		return Extraction{}, fmt.Errorf("parsing extraction: %w", err)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return Extraction{}, errors.New("parsing extraction: content after the JSON object")
	}
	if a.Entity == "" {
		return Extraction{}, errors.New("parsing extraction: entity is missing")
	}
	for i, c := range a.Conditions {
		// The schema cannot tie value to the operator, so a model sometimes sends "value": "" with exists.
		// The operator takes no value, so dropping it changes nothing the request asked for.
		if rule, ok := c.Operator.Rule(); ok && rule.Arity == ir.ArityNone {
			a.Conditions[i].Value = nil
		}
	}
	return Extraction{
		Query:         ir.Query{Entity: a.Entity, Conditions: a.Conditions},
		UnmappedTerms: a.UnmappedTerms,
	}, nil
}

func attributeKind(a dynamoschema.Attribute) string {
	if a.Format == "" {
		return a.Type
	}
	return a.Type + ", " + a.Format
}

func tableNames(tables []dynamoschema.Table) []string {
	names := make([]string, len(tables))
	for i, t := range tables {
		names[i] = t.Name
	}
	return names
}

// fieldNames is the union of every table's attributes in first-seen order; the validator rejects a field used
// on the wrong table.
func fieldNames(tables []dynamoschema.Table) []string {
	names := []string{}
	seen := map[string]bool{}
	for _, t := range tables {
		for _, a := range t.Attributes {
			if !seen[a.Name] {
				seen[a.Name] = true
				names = append(names, a.Name)
			}
		}
	}
	return names
}
