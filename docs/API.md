# Sagasu API Reference

REST API for the Sagasu hybrid search engine.

Base URL: `http://localhost:8080` (default)

## Endpoints

### POST /api/v1/search

Run a hybrid (keyword + semantic) search.

**Request body:**

```json
{
  "query": "machine learning",
  "limit": 10,
  "offset": 0,
  "keyword_weight": 0.5,
  "semantic_weight": 0.5,
  "min_score": 0.0
}
```

| Field           | Type   | Description                        |
| --------------- | ------ | ---------------------------------- |
| query           | string | Required. Search query text.       |
| limit           | int    | Max results (default 10, max 100). |
| offset          | int    | Pagination offset (default 0).     |
| keyword_weight  | float  | Weight for keyword score (0–1).    |
| semantic_weight | float  | Weight for semantic score (0–1).   |
| min_score       | float  | Minimum fused score to include.    |

**Response (200):**

```json
{
  "results": [
    {
      "document": {
        "id": "doc-id",
        "title": "Document Title",
        "content": "...",
        "metadata": {},
        "created_at": "...",
        "updated_at": "..."
      },
      "score": 0.85,
      "keyword_score": 0.9,
      "semantic_score": 0.8,
      "rank": 1
    }
  ],
  "total": 1,
  "query_time_ms": 25,
  "query": "machine learning"
}
```

**Errors:** 400 (invalid body), 500 (search failure).

---

### POST /api/v1/documents

Index a document.

**Request body:**

```json
{
  "id": "optional-id",
  "title": "Document Title",
  "content": "Full text content to index.",
  "metadata": {}
}
```

| Field    | Type   | Description                          |
| -------- | ------ | ------------------------------------ |
| id       | string | Optional. Auto-generated if omitted. |
| title    | string | Optional title.                      |
| content  | string | Required. Body text.                 |
| metadata | object | Optional key-value metadata.         |

**Response (201):**

```json
{
  "id": "doc-id",
  "status": "indexed"
}
```

**Errors:** 400 (invalid body), 500 (indexing failure).

---

### GET /api/v1/documents/{id}

Fetch a document by ID.

**Response (200):** Document JSON (same shape as in search results).

**Errors:** 404 (not found).

---

### DELETE /api/v1/documents/{id}

Delete a document and remove it from all indices.

**Response (200):**

```json
{
  "status": "deleted"
}
```

**Errors:** 500 (deletion failure).

---

### GET /health

Health check.

**Response (200):**

```json
{
  "status": "ok"
}
```

---

## Error format

Error responses use JSON:

```json
{
  "error": "Human-readable error message"
}
```

HTTP status codes: 400 Bad Request, 404 Not Found, 500 Internal Server Error.
