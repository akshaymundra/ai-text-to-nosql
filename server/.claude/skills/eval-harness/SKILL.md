---
name: eval-harness
description: How to add, run and interpret evaluation cases that score the text-to-DynamoDB pipeline end to end (Ollama extraction, planner, validator, compiler). Use when adding an eval case, fixing a bug found in a query, changing the extraction prompt, the Ollama model or the structured-output schema, or reporting accuracy.
---

# Eval harness

The eval set is the only proof that the LLM stage works. Unit tests cover the deterministic code; evals cover "English in, correct `QueryInput` or correct rejection out".

## Layout and format

One JSON file per case in `server/evals/cases/`, named `<nnn>-<slug>.json`:

```json
{
  "id": "014-active-acme-users-last-month",
  "request": "Get active users from acme who signed up last month",
  "now": "2026-10-07T12:00:00Z",
  "tags": ["gsi", "filter", "relative-date"],
  "expect": {
    "status": "valid",
    "table": "users",
    "index": "organization-created-at-index",
    "query": {
      "KeyConditionExpression": "#n0 = :v0 AND #n1 BETWEEN :v1 AND :v2",
      "FilterExpression": "#n2 = :v3",
      "ExpressionAttributeNames": { "#n0": "organization_id", "#n1": "created_at", "#n2": "status" },
      "ExpressionAttributeValues": { ":v0": "acme", ":v1": "2026-09-01T00:00:00Z", ":v2": "2026-09-30T23:59:59Z", ":v3": "active" }
    }
  }
}
```

A rejection case:

```json
{
  "id": "021-email-contains-gmail",
  "request": "Find users whose email contains gmail",
  "now": "2026-10-07T12:00:00Z",
  "tags": ["reject", "scan"],
  "expect": { "status": "rejected", "code": "SCAN_REQUIRED" }
}
```

Rules:
- `ExpressionAttributeValues` are written as plain JSON values; the harness unmarshals the compiled `AttributeValue`s before comparing. Comparison is exact, which relies on the compiler's deterministic placeholder naming.
- `now` is fixed per case. Relative dates resolve against it, never the wall clock.
- Rejection `code` must be a code the validator defines in `internal/constants`. Do not invent codes in a case; add the code to the validator first.
- Expected values come from data in `server/schemas-local/` so a valid case also returns rows when executed against DynamoDB Local.
- Mix: valid table-key queries, valid GSI queries, filtered queries, unknown fields, scan-required requests, invalid sort-key operators, relative dates. Target 30 to 50 cases.

## Running

```sh
make eval            # needs local Ollama running with the configured model
```

- Prints one line per case (`PASS`/`FAIL`, id, and on failure the extracted IR plus expected vs actual) and a total score.
- On failure, the extracted IR tells you where it broke: wrong IR = extraction (prompt/model); right IR but wrong query = planner/compiler bug, which needs a unit test too.
- Ollama settings are part of the result: pinned model tag, `temperature: 0`, fixed `seed`. Structured output via the `format` JSON schema works with **local Ollama only**; Ollama Cloud does not support it, so cloud runs score extraction without schema enforcement.

## Workflow rules

1. **Bug in a query = eval case first.** Reproduce it as a failing case, then fix.
2. **Any change to the extraction prompt, `format` schema, model or Ollama options:** run `make eval` before and after, and report both scores and which cases flipped. A change that fixes one case and breaks two is a regression.
3. **Never edit an `expect` block to make a case pass** unless the user agrees the expectation was wrong. Say which case and why.
4. Keep cases independent; no case relies on another's output.
5. Do not tune the prompt with the exact wording of eval requests. Add new phrasings as new cases instead, so the set measures generalisation rather than memorisation.
