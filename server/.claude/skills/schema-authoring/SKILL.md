---
name: schema-authoring
description: How to write or change a table schema JSON file (server/schemas-local/*.json, and later workspace schema annotations) so seeding, retrieval, LLM extraction and planning all work. Use when adding a demo table, adding an attribute, index or access pattern, or writing descriptions that retrieval and the LLM will read.
---

# Schema authoring

One file per table in `server/schemas-local/`. Loaded by `internal/dynamoschema.LoadDir`, seeded by `make seed`, and (later) embedded for retrieval. The JSON shape is the `dynamoschema.Table` struct; check it before adding fields.

## Contract

```json
{
  "table": "users",
  "description": "People who belong to an organization and can log in to the product.",
  "partitionKey": "user_id",
  "sortKey": "",
  "attributes": [
    { "name": "created_at", "type": "string", "description": "Signup time, UTC, format YYYY-MM-DDTHH:MM:SSZ." }
  ],
  "indexes": [
    { "name": "organization-created-at-index", "partitionKey": "organization_id", "sortKey": "created_at" }
  ],
  "accessPatterns": [
    { "description": "List users of an organization, optionally within a signup date range.", "index": "organization-created-at-index" }
  ],
  "exampleRecords": [ { "user_id": "u_001", "organization_id": "acme", "created_at": "2025-03-14T10:00:00Z" } ]
}
```

## Rules

**Keys and indexes**
- Every key attribute (table and index PK/SK) must be listed in `attributes` with type `string` or `number`; the seeder rejects anything else.
- One partition key and at most one sort key per index. DynamoDB now supports multi-attribute GSI keys, but `dynamoschema.Table` does not model them; do not add one.
- Table/index names: 3 to 255 chars, `a-z A-Z 0-9 _ - .`.
- GSIs are created with projection `ALL`, so every attribute is queryable through any index. If projections change later, the schema must state them, because a GSI cannot return non-projected attributes.

**Attributes and descriptions** (read by the LLM and by retrieval)
- Description says what the value means **and its format/allowed values**: `"Account status: active, invited or disabled."` The LLM copies values literally, so list the exact stored spellings.
- Dates: one fixed-width UTC format per attribute, stated in the description: `YYYY-MM-DDTHH:MM:SSZ` for timestamps, `YYYY-MM-DD` for pure dates (e.g. `due_date`). The compiler builds range bounds in that format; mixed formats break range queries (see `dynamodb-query-rules`, section 7).
- Use business words a user would say ("signed up", "owner", "due") in descriptions, because names like `created_at` are not what people type.
- Reserved words (`status`, `name`, `date`, ...) are fine as attribute names; the compiler always aliases.

**Access patterns** (one embedding row each)
- One pattern per distinct question shape, phrased the way a user would ask it: "List tasks of a project by due date", not "Query GSI2 on project_id".
- `index` names the GSI that serves it; omit for the table's own key.
- Every index has at least one access pattern, and every pattern is servable by its key.

**Example records**
- Every access pattern must have records that match it, including the edge of each range (e.g. a record on the last day of a month), so eval cases return rows.
- Values follow the declared formats exactly; numbers as JSON numbers, not strings.
- Include at least one record missing an optional GSI key attribute when sparse-index behaviour matters.

## After editing

1. `make seed` from `server/` (needs DynamoDB Local; drops and recreates every table).
2. `make test`.
3. If attributes, values or patterns changed, update or add eval cases (`eval-harness` skill) and run `make eval`.
