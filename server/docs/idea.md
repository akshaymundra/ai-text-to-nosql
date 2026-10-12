# AI-Powered NoSQL Query Grounding

## 1. Project Objective

Build an AI system that converts natural-language requests into **valid, schema-grounded DynamoDB queries**.

The system must not blindly generate database queries. It should first understand the user's intent, retrieve the relevant schema and access-pattern information, generate a structured query representation, validate it against the schema, and only then compile it into a DynamoDB query.

The primary goal of Phase 1 is to prove that **LLMs can generate reliable NoSQL queries when constrained by real schema and access-pattern rules**.

This is a personal learning project. Technology choices favour hands-on practice (for example a real vector store) over the leanest possible build.

### Core Principle

The LLM is only a parser: it turns English into a structured intermediate representation (IR). Every decision that matters (which index to use, whether a scan is required, whether the query is valid) is made by deterministic code. The validator and compiler are the product.

---

## 2. Phase 1 Scope

### A. Schema Metadata

Create a small set of DynamoDB schemas, approximately 5-10 tables.

Each schema should define:

- Table name
- Attributes
- Partition key
- Sort key
- GSIs
- Index partition/sort keys
- Attribute types
- Supported access patterns
- Example records

Example:

```json
{
  "table": "users",
  "partitionKey": "tenant_id",
  "sortKey": "created_at",
  "indexes": [
    {
      "name": "organization-created-at-index",
      "partitionKey": "organization_id",
      "sortKey": "created_at"
    }
  ]
}
```

### B. Schema Retrieval

Implement a lightweight retrieval layer that receives the user's request and retrieves the most relevant tables, indexes, and access-pattern metadata.

Retrieval runs **before** intent extraction, because the LLM cannot name the correct fields without seeing the schema.

Retrieval uses **pgvector** (Postgres). Embeddings are stored at the access-pattern level (one row per table, index and access pattern, with its description), not one row per table, so semantic search has meaningful granularity even with a small number of tables. Each row keeps a reference to the original structured schema, which is what gets passed downstream.

### C. Intent Extraction

Convert the natural-language request into a structured intermediate representation instead of directly generating DynamoDB syntax.

Rules:

- The LLM receives only the retrieved schema and must answer through structured output (tool use / JSON Schema). Table and field names are enums built from the retrieved schema, which prevents most hallucinated fields before validation.
- The IR contains the entity and conditions only. It does **not** name an index. Index selection is the planner's job.
- The current date is injected into the prompt so relative dates ("last month") resolve to absolute values.
- Values are taken literally. Resolving names to IDs (for example "Acme" to `org_7f3a...`) is out of scope for Phase 1.

Example:

```json
{
  "entity": "users",
  "conditions": [
    {
      "field": "organization_id",
      "operator": "eq",
      "value": "acme"
    },
    {
      "field": "created_at",
      "operator": "between",
      "value": ["2026-09-01", "2026-09-30"]
    },
    {
      "field": "status",
      "operator": "eq",
      "value": "active"
    }
  ]
}
```

### D. Access-Path Planner and Deterministic Validator

The planner chooses the access path from the IR conditions, without any LLM involvement:

1. Find the table or GSI whose partition key has an `eq` condition.
2. Check that the sort-key condition (if any) uses an operator DynamoDB supports on sort keys (`eq`, `lt`, `lte`, `gt`, `gte`, `between`, `begins_with`).
3. Move all remaining conditions to `FilterExpression`.
4. If no table key or index fits, reject the request as a scan.

Because the planner picks the index, "this would require a scan" is a fact proven by code, not a judgement made by the LLM.

Queries that use a `FilterExpression` are valid but still consume read capacity for every item the key condition matches. They are returned as `valid` with a warning.

The validator also checks:

- Table exists
- Fields exist
- Index exists
- Partition-key requirements are satisfied
- Sort-key operators are valid
- Requested operation is compatible with DynamoDB Query rules
- Unsupported access patterns are rejected

The validator should be able to reject an AI-generated query even when the JSON format itself is valid.

### E. DynamoDB Query Compiler

Convert the validated intermediate representation into an executable DynamoDB query.

For example:

```text
Natural language
      ↓
Schema retrieval
      ↓
Intent extraction (IR)
      ↓
Access-path planning
      ↓
Validation
      ↓
DynamoDB QueryInput
```

### F. REST API

Expose a simple API:

```text
POST /api/v1/queries
```

Request:

```json
{
  "query": "Get active users from Acme who signed up last month"
}
```

Response:

```json
{
  "status": "valid",
  "table": "users",
  "index": "organization-created-at-index",
  "query": {
    "KeyConditionExpression": "...",
    "FilterExpression": "..."
  },
  "warnings": [
    "status is applied as a FilterExpression; read capacity is consumed for all items matching the key condition"
  ]
}
```

`index` is the planner's output, not the LLM's.

### G. Failure Handling

The system must clearly reject requests when the schema cannot support them.

Example:

```text
Request:
Find users by email contains "gmail"

Result:
Rejected

Reason:
No available partition key or index supports this access pattern.
Executing this request would require a table scan.
```

---

## 3. Phase 1 Architecture

```text
                 User
                  │
                  ▼
          Natural Language Query
                  │
                  ▼
     Schema / Pattern Retrieval (pgvector)
                  │
                  ▼
     Intent Extraction LLM (structured output)
                  │
                  ▼
        Structured Query Representation (IR)
                  │
                  ▼
         Access-Path Planner (no LLM)
                  │
                  ▼
        Deterministic Query Validator
             │              │
          Valid           Invalid
             │              │
             ▼              ▼
      Query Compiler     Explanation
             │
             ▼
       DynamoDB Query
```

---

## 4. Recommended Technology

- **Backend:** Go or Node.js
- **Database:** DynamoDB
- **Vector Store:** Postgres with pgvector
- **LLM:** OpenAI or Anthropic
- **Embeddings:** OpenAI embeddings or local embedding model
- **API:** REST
- **Testing:** Unit tests + integration tests
- **Deployment:** Docker

For Phase 1, use a small local/sandbox DynamoDB environment rather than building production infrastructure.

---

## 5. Phase 1 Success Criteria

The project is successful when it can:

1. Understand a natural-language database request.
2. Retrieve the correct schema and access pattern.
3. Produce a structured query representation.
4. Reject nonexistent fields, indexes, and unsupported access patterns.
5. Generate valid DynamoDB Query operations.
6. Refuse requests that require unsafe or unsupported table scans.
7. Execute valid queries against a test DynamoDB dataset.
8. Explain why a generated query was accepted or rejected.

Reliability is measured with an **evaluation set**: 30-50 natural-language requests, each with an expected outcome (expected QueryInput or expected rejection reason). The set mixes valid queries, nonexistent fields, scan-requiring requests, and invalid sort-key operators, and is run after every change to the prompt or pipeline.

## Build Order

1. Schema metadata files for 5-8 tables with realistic access patterns; create and seed them in DynamoDB Local.
2. IR types, planner, validator and compiler: pure code, test-first, no LLM.
3. Evaluation set.
4. pgvector retrieval over access-pattern chunks.
5. LLM intent extraction with structured output, scored against the evaluation set.
6. `POST /api/v1/queries` and execution against DynamoDB Local.
7. Human-readable explanations built from the validator's structured reasons.

---

## 6. Explicitly Out of Scope for Phase 1

The following should not be built initially:

- MongoDB support
- PostgreSQL/SQL support
- Multi-table joins or complex cross-service queries
- Automatic index creation
- Query cost prediction
- Query optimization
- Production database access
- Autonomous database modifications
- Multi-agent architecture
- Organization-wide knowledge graph
- Fine-tuning models
- Automatic schema discovery from live production systems
- Complex authentication/authorization
- Production deployment infrastructure

---

# Future Scope

Once Phase 1 works reliably, the system can evolve into a broader **AI Database Gateway**.

Potential future capabilities include:

- Query cost and performance analysis
- Query optimization
- Automatic index recommendations
- Multi-table reasoning
- MongoDB/PostgreSQL support
- Schema discovery from real databases
- Tenant-aware query enforcement
- Query security policies
- Explainable query plans
- Query history and feedback learning
- Production observability
- Database migration assistance
- AI-powered database architecture recommendations
- Natural-language analytics
- Organization-wide data knowledge graph

## Long-Term Vision

The long-term goal is not simply:

**"Convert English to DynamoDB."**

It is:

**"Create an AI layer that understands database schemas, access patterns, constraints, and safety rules, and can reliably reason about how data should be accessed."**