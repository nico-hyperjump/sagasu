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

Results are split into two disjoint lists: `non_semantic_results` (keyword-only) and `semantic_results` (semantic-only). No document appears in both.

```json
{
  "non_semantic_results": [
    {
      "document": {
        "id": "doc-id",
        "title": "Document Title",
        "content": "...",
        "metadata": {},
        "created_at": "...",
        "updated_at": "..."
      },
      "score": 0.9,
      "keyword_score": 0.9,
      "semantic_score": 0,
      "rank": 1
    }
  ],
  "semantic_results": [
    {
      "document": {
        "id": "doc-id-2",
        "title": "Another Document",
        "content": "...",
        "metadata": {},
        "created_at": "...",
        "updated_at": "..."
      },
      "score": 0.8,
      "keyword_score": 0,
      "semantic_score": 0.8,
      "rank": 1
    }
  ],
  "total_non_semantic": 1,
  "total_semantic": 1,
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

### GET /api/v1/watch/directories

List watched directories (directories monitored for file changes).

**Response (200):**

```json
{
  "directories": ["/path/to/docs", "/path/to/notes"]
}
```

**Errors:** 501 (watch not enabled).

---

### POST /api/v1/watch/directories

Add a directory to watch. Files matching configured extensions are indexed on create/modify and removed from the index when deleted.

**Request body:**

```json
{
  "path": "/absolute/path/to/directory",
  "sync": true
}
```

| Field | Type   | Description                                    |
| ----- | ------ | ---------------------------------------------- |
| path  | string | Required. Absolute path to directory.          |
| sync  | bool   | Optional. Index existing files (default true). |

**Response (201):**

```json
{
  "path": "/absolute/path/to/directory",
  "status": "added"
}
```

**Errors:** 400 (invalid path), 404 (directory not found), 500 (add failure).

---

### DELETE /api/v1/watch/directories

Remove a directory from watching. Does not delete documents already indexed from that path.

**Query parameter or JSON body:**

| Field | Type   | Description               |
| ----- | ------ | ------------------------- |
| path  | string | Required. Path to remove. |

**Response (200):**

```json
{
  "path": "/absolute/path/to/directory",
  "status": "removed"
}
```

**Errors:** 400 (path required), 500 (remove failure).

---

### GET /api/v1/status

Return engine, storage, and index statistics. All numeric fields are counts unless otherwise noted.

**Response (200):**

```json
{
  "documents": 42,
  "chunks": 150,
  "vector_index_size": 150,
  "disk_usage_bytes": 1048576
}
```

| Field             | Type | Description                                                                 |
| ----------------- | ---- | --------------------------------------------------------------------------- |
| documents         | int  | Count of documents in storage.                                              |
| chunks            | int  | Count of text chunks in storage.                                            |
| vector_index_size | int  | Count of vectors in the semantic index (one per chunk).                     |
| disk_usage_bytes  | int  | Optional. Total bytes used on disk by the database and index paths (bytes). |

**Errors:** 500 (storage or count failure).

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
