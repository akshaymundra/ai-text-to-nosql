---
name: eval-harness
description: How to add, run and interpret evaluation cases that score the text-to-DynamoDB pipeline end to end (Ollama extraction, planner, validator, compiler). Use when adding an eval case, fixing a bug found in a query, changing the extraction prompt, the Ollama model or the structured-output schema, or reporting accuracy.
---

# Eval harness

The eval set is the only proof that the LLM stage works. Unit tests cover the deterministic code; evals cover "English in, correct `QueryInput` or correct rejection out".

## Layout and format

Go package `server/evals/` (`evals.go`: `Case`, `Outcome`, `LoadCases`, `OutcomeOf`). One JSON file per case in `server/evals/cases/`, named `<nnn>-<slug>.json`; unknown keys fail loading, so a typo cannot pass silently.

```json
{
  "id": "001-active-acme-users-last-month",
  "request": "Get active users from acme who signed up last month",
  "now": "2026-10-07T12:00:00Z",
  "tags": ["gsi", "filter", "relative-date"],
  "ir": {
    "entity": "users",
    "conditions": [
      {"field": "organization_id", "operator": "eq", "value": "acme"},
      {"field": "created_at", "operator": "between", "value": ["2026-09-01", "2026-09-30"]},
      {"field": "status", "operator": "eq", "value": "active"}
    ]
  },
  "expect": {
    "status": "valid",
    "table": "users",
    "index": "organization-created-at-index",
    "query": {
      "KeyConditionExpression": "#n0 = :v0 AND #n1 BETWEEN :v1 AND :v2",
      "FilterExpression": "(#n2 = :v3)",
      "ExpressionAttributeNames": {"#n0": "organization_id", "#n1": "created_at", "#n2": "status"},
      "ExpressionAttributeValues": {":v0": "acme", ":v1": "2026-09-01T00:00:00Z", ":v2": "2026-09-30T23:59:59Z", ":v3": "active"}
    }
  }
}
```

A rejection case (`field` only when the rejection names one; `SCAN_REQUIRED` does not):

```json
"expect": {"status": "rejected", "code": "INVALID_KEY_CONDITION", "field": "project_id"}
```

Rules:
- `ir` is the reference IR: what a correct extraction produces, with relative dates resolved against `now` and dates left as written (`"2026-09-01"`; the `dates` stage widens them). For a rejection case it is the honest IR of the request, which the pipeline then rejects. Never write an IR that only a wrong extraction would produce.
- A request naming a field no table has ("users by phone number") lists it in `"unmapped_terms": ["phone number"]` next to `ir`; `BuildQuery` turns each term into `UNKNOWN_FIELD` with that term as `field`. Suggestions (`"did you mean name?"`) depend on embeddings, so `expect` never includes them.
- So no case expects `UNKNOWN_TABLE` (extraction enums make it impossible) or `MULTIPLE_KEY_CONDITIONS` (the honest IR for a range is one `between`); unit tests cover those codes.
- `expect` is written by hand from the planner and compiler rules, never copied from pipeline output, or the case proves nothing.
- `ExpressionAttributeValues` are plain JSON values; `OutcomeOf` converts the compiled `{"S": ...}` / `{"N": ...}` before comparing. Filters are each parenthesised. Comparison is exact and relies on deterministic placeholder naming. `Limit` and warnings are not compared.
- Only the first rejection is compared (`code`, and `field` when present).
- Rejection `code` must exist in `internal/constants`. Do not invent codes in a case.
- `now` is fixed per case (all current cases use `2026-10-07T12:00:00Z`). Relative dates resolve against it, never the wall clock.
- Values come from `server/schemas-local/` data, so a valid case also returns rows when executed against DynamoDB Local.
- Tags name what a case exercises (`gsi`, `table`, `sort-key`, `filter`, `date-range`, `relative-date`, `tie-break`, `reject`, `scan`, ...), so scores can be reported per feature. Reuse existing tags before adding new ones.
- Current set: `001`-`030` valid, `031`-`046` rejected. Target 30 to 50 cases; grow it when a bug or a new feature needs one.

## Running

```sh
make test                       # includes go test ./evals: reference IR -> BuildQuery -> compare with expect, no LLM
go test -run TestCases -v ./evals   # per-case PASS/FAIL
make eval                       # step 5, not built yet: request -> Ollama extraction -> BuildQuery -> compare
```

- `TestCases` proves every `expect` against the real schemas before any LLM is involved. A failure there is a wrong case or a pipeline regression, never the model.
- `make eval` (planned) reuses `LoadCases` and `OutcomeOf`, and also diffs the extracted IR against `ir`: wrong IR = extraction (prompt/model); right IR but wrong query = planner/compiler bug, which needs a unit test too. It prints one line per case, a total score and a score per tag.
- Ollama settings are part of the result: pinned model tag, `temperature: 0`, fixed `seed`. Structured output via the `format` JSON schema works with **local Ollama only**; Ollama Cloud does not support it, so cloud runs score extraction without schema enforcement.

## Workflow rules

1. **Bug in a query = eval case first.** Reproduce it as a failing case, then fix.
2. **Any change to the extraction prompt, `format` schema, model or Ollama options:** run `make eval` before and after, and report both scores and which cases flipped. A change that fixes one case and breaks two is a regression.
3. **Never edit an `expect` block to make a case pass** unless the user agrees the expectation was wrong. Say which case and why.
4. Keep cases independent; no case relies on another's output.
5. Do not tune the prompt with the exact wording of eval requests. Add new phrasings as new cases instead, so the set measures generalisation rather than memorisation.
