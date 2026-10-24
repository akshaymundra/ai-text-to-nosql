---
name: dynamodb-query-rules
description: Exact DynamoDB Query semantics (key conditions, filter expressions, placeholders, GSI behaviour, pagination, limits) verified against AWS docs. Use when writing, testing or reviewing the IR operators, access-path planner, validator, compiler or query execution, or when answering any question about what a DynamoDB Query can and cannot do.
---

# DynamoDB Query rules

Verified against the AWS docs on 2026-10-07. When a rule here conflicts with memory, this file wins. When AWS docs conflict with this file, re-fetch the source link, fix this file, and tell the user.

The planner, validator and compiler must encode these rules as code and tests. Never rely on the LLM for any of them.

## 1. Key condition (`KeyConditionExpression`)

Source: [API_Query](https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_Query.html), [Query.KeyConditionExpressions](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Query.KeyConditionExpressions.html)

- Partition key: **required, equality only** (`pk = :v`), exactly one value.
- Sort key: **optional, at most one condition**, joined with `AND`. Allowed forms only:
  `=`, `<`, `<=`, `>`, `>=`, `BETWEEN :a AND :b` (inclusive both ends), `begins_with(sk, :v)`.
- `begins_with` is case-sensitive and **not allowed on a Number sort key**.
- Not allowed in a key condition: `<>`, `OR`, `NOT`, `IN`, `contains`, `attribute_exists`, `size`, any non-key attribute.
- No condition on the sort key is valid: it reads the whole partition (item collection).

## 2. Filter expression (`FilterExpression`)

Source: [Query.FilterExpression](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Query.FilterExpression.html), [Expressions.OperatorsAndFunctions](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Expressions.OperatorsAndFunctions.html)

- **Must not contain the partition key or sort key** of the table/index being queried. A condition on a key attribute that cannot go in the key condition (e.g. `sk <> :v`) cannot go in the filter either: reject it.
- Applied **after** items are read. Read capacity is the same with or without a filter. This is why filtered queries return `valid` with a warning.
- Allowed: comparators `= <> < <= > >=`, `BETWEEN`, `IN (...)` (max **100** operands), `AND`, `OR`, `NOT`, parentheses, and functions `attribute_exists(path)`, `attribute_not_exists(path)`, `attribute_type(path, :t)`, `begins_with(path, :v)`, `contains(path, :v)`, `size(path)`.
- `contains`: string substring, or element of a set/list. Operand type must match (string for string, element type for set). `contains(a, a)` is an error.
- Boolean attributes must be compared (`#v = :true`); a bare attribute is not a condition.
- Function names are case-sensitive.
- Precedence, high to low: comparators, `IN`, `BETWEEN`, functions, parentheses, `NOT`, `AND`, `OR`. The compiler should parenthesise explicitly instead of relying on this.

## 3. Placeholders

Source: [Expressions.ExpressionAttributeNames](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Expressions.ExpressionAttributeNames.html), [Expressions.ExpressionAttributeValues](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Expressions.ExpressionAttributeValues.html)

- Name placeholders: `#` + alphanumerics/underscore. Value placeholders: `:` + alphanumerics.
- Placeholders are required for reserved words, names starting with a digit, and names containing space, `.` or `-`. A dot in a raw name means a nested path.
- **Project rule: the compiler aliases every attribute name and every value**, so reserved words never matter. Many demo attributes are reserved (`status`, `name`, `role`, `date`, `user`, `type`, `count`, `region`). Full list: [references/reserved-words.txt](references/reserved-words.txt).
- Placeholder names must be deterministic (e.g. `#n0, #n1` / `:v0, :v1` in condition order) so eval cases can compare compiled output exactly.
- Only emit placeholders that the expressions use; unused entries in `ExpressionAttributeNames`/`Values` make the request fail validation.
- Values are typed `AttributeValue`s: `S`, `N` (number sent as a string), `BOOL`, `NULL`, `B`, sets, `L`, `M`. Never inline literals in expressions.

## 4. Request parameters the compiler emits

Source: [API_Query](https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_Query.html)

| Param | Rule |
|---|---|
| `TableName` | Always. |
| `IndexName` | Only when the planner chose a GSI. Set by the planner, never by the LLM. |
| `KeyConditionExpression` | Always (section 1). |
| `FilterExpression` | Only when non-key conditions remain. |
| `ExpressionAttributeNames` / `Values` | Section 3. |
| `Limit` | Max items **evaluated** (not returned), min 1. Applied before the filter. |
| `ExclusiveStartKey` | Cursor from a previous `LastEvaluatedKey`. Key types S/N/B only. |
| `ScanIndexForward` | `false` for descending sort-key order ("latest first"). Default ascending. |
| `ConsistentRead` | **Never on a GSI** (`ValidationException`). Omit by default. |
| `ProjectionExpression` / `Select` | `Select` other than `SPECIFIC_ATTRIBUTES` cannot be combined with a projection. `COUNT` costs the same capacity as reading items. |
| Legacy (`KeyConditions`, `QueryFilter`, `AttributesToGet`, `ConditionalOperator`) | Never emit. |

## 5. Pagination and capacity

- One Query reads at most **1 MB** (before filter and projection) or `Limit` items, whichever first.
- `LastEvaluatedKey` present means "maybe more". A page can be **empty with a `LastEvaluatedKey`** when the filter dropped everything read. Only an absent key means done.
- `Count` = items returned after the filter; `ScannedCount` = items read. A high ratio means a wasteful filter.

## 6. GSI specifics

Source: [GSI](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/GSI.html)

- Key attributes: top-level `S`, `N` or `B` only.
- Keys are **not unique** in a GSI. Equal sort keys come back in no particular order.
- **Sparse:** items missing an index key attribute are not in the index. A GSI query can return fewer items than the base table holds.
- **Eventually consistent only.** Fresh writes may be missing.
- Projection `KEYS_ONLY` / `INCLUDE` / `ALL`: a GSI query **cannot fetch non-projected attributes** from the base table. If a filter or projection needs an attribute the index does not project, that index cannot serve the request.
- **Multi-attribute keys (new):** a GSI can have up to 4 partition-key and 4 sort-key attributes. Query must give `=` on all partition attributes; sort attributes left to right with no gaps; an inequality (`< > BETWEEN begins_with`) only on the last one used. `internal/dynamoschema` models a single PK/SK today: reject or flag such indexes until the schema model supports them.
- Default quota: 20 GSIs per table.

## 7. Types, ordering, limits

Source: [HowItWorks.NamingRulesDataTypes](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/HowItWorks.NamingRulesDataTypes.html), [Constraints](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Constraints.html)

- Number: up to 38 digits, sorted numerically. String/Binary: sorted by UTF-8 / unsigned bytes ("a" > "A").
- No date type. Dates are ISO 8601 strings (lexicographic) or epoch numbers.
- **Date-range trap:** with ISO timestamps, `BETWEEN "2026-09-01" AND "2026-09-30"` **excludes** `2026-09-30T10:00:00Z`, because that string sorts after `"2026-09-30"`. A sort key allows only one condition, so `>= start AND < end` is not available. Rule: each date attribute uses one fixed-width UTC format declared by its schema `format` field (`timestamp` = `YYYY-MM-DDTHH:MM:SSZ`, `date` = `YYYY-MM-DD`), and the `dates` stage (`internal/service/dates`) rewrites bounds into that attribute's format: "in September" = `BETWEEN "2026-09-01T00:00:00Z" AND "2026-09-30T23:59:59Z"` for a timestamp, `BETWEEN "2026-09-01" AND "2026-09-30"` for a date. Mixed formats (with/without millis, offsets) break lexicographic order; `"...59.5Z"` sorts before `"...59Z"`. Resolve relative dates against the injected "now", in UTC.
- Key values cannot be empty strings. Partition key max 2048 bytes, sort key max 1024 bytes.
- Expression string max **4 KB**; each placeholder max 255 bytes; all substitutions max 2 MB; max **300** operators/functions per expression; `IN` max 100 operands.
- Table/index names: 3 to 255 chars of `a-z A-Z 0-9 _ - .`.

## 8. IR operator mapping

| IR op | Partition key | Sort key | Non-key (filter) |
|---|---|---|---|
| `eq` | yes (required) | yes | `=` |
| `lt` `lte` `gt` `gte` | no | yes | `< <= > >=` |
| `between` | no | yes (inclusive) | `BETWEEN` |
| `begins_with` | no | yes, String only | `begins_with` |
| `ne`, `in`, `contains`, `exists`, `not_exists` | no | **no, and not in filter either** | `<>`, `IN`, `contains`, `attribute_exists`, `attribute_not_exists` |

Planner outcome per candidate (table or each GSI): usable only if the IR has an `eq` on its partition key and at most one sort-key-compatible condition on its sort key; every other condition must be filter-legal and on a projected attribute. If no candidate is usable, reject as a scan.

## 9. Test checklist for planner/validator/compiler

- Partition key with `eq` vs any other operator.
- Two conditions on the same sort key (e.g. `gte` + `lte`): merge into `BETWEEN` or reject. Pick one rule and test it.
- `begins_with` on a Number sort key.
- `ne` / `in` / `contains` on a key attribute.
- Filter on a non-projected attribute of a `KEYS_ONLY` / `INCLUDE` GSI.
- Several usable indexes: deterministic tie-break (e.g. prefer the one that also uses a sort-key condition, then schema order).
- Reserved-word attribute names compile with placeholders.
- `IN` with more than 100 values.
- ISO date range covering a whole day or month.
- `ConsistentRead` never set on a GSI query.
