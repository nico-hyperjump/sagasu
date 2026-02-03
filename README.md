# Sagasu

Fast local hybrid search engine combining semantic and keyword search for macOS.

## Features

- **Fast**: Sub-50ms query times for 100k documents (with FAISS/ONNX).
- **Hybrid search**: Combines semantic (embeddings) and keyword (BM25-style) search.
- **Low memory**: In-memory vector index; optional FAISS for scale.
- **100% local**: No external APIs or cloud services.
- **Private**: All data stays on your machine.
- **Simple**: CLI and HTTP API.

## Installation

### Homebrew (macOS) — production build

```bash
brew tap hyperjump/sagasu
brew install sagasu
```

The formula depends on ONNX Runtime and builds with CGO, so you get real semantic embeddings. After install it downloads the embedding model (~80MB) and sets up config and data dirs (e.g. under `/usr/local/etc/sagasu` and `/usr/local/var/sagasu` on Intel; paths may differ on Apple Silicon).

### From source — production build

Prerequisites: Go 1.21+, ONNX Runtime (e.g. `brew install onnxruntime` on macOS), and the embedding model.

```bash
git clone https://github.com/hyperjump/sagasu.git
cd sagasu

# Build with CGO (links to ONNX Runtime)
make build

# Download the embedding model (one-time, ~80MB)
./scripts/download-model.sh /usr/local/var/sagasu/data/models

# Install binary and config
make install
```

This installs the binary to `/usr/local/bin/sagasu`, copies `config.yaml` to `/usr/local/etc/sagasu/`, and creates data dirs under `/usr/local/var/sagasu/data/`. Start the server with `sagasu server` or `brew services start sagasu` if you used Homebrew.

For a **development build** without ONNX (mock embedder, in-memory index): `CGO_ENABLED=0 go build -o bin/sagasu ./cmd/sagasu`.

## Quick start

### Start the server

```bash
# As a service (background)
brew services start sagasu

# Or run directly
sagasu server
```

### Index documents

```bash
sagasu index --title "My Document" document.txt
```

Or via HTTP:

```bash
curl -X POST http://localhost:8080/api/v1/documents \
  -H "Content-Type: application/json" \
  -d '{"title": "Machine Learning Basics", "content": "Machine learning is a subset of AI..."}'
```

### Search

```bash
sagasu search "machine learning algorithms"
sagasu search --keyword-weight 0.7 --semantic-weight 0.3 "neural networks"
```

Or via HTTP:

```bash
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "machine learning", "limit": 10}'
```

## Configuration

Config file: `/usr/local/etc/sagasu/config.yaml` (or path set with `--config`).

See `config.yaml.example` for server, storage, embedding, and search options (host, port, paths, chunk size, weights, etc.).

Restart the server after changing config:

```bash
brew services restart sagasu
```

## Data locations

- **Config**: `/usr/local/etc/sagasu/config.yaml`
- **Database**: `/usr/local/var/sagasu/data/db/`
- **Indices**: `/usr/local/var/sagasu/data/indices/`
- **Models**: `/usr/local/var/sagasu/data/models/`

## API and CLI docs

- [API reference](docs/API.md) – REST endpoints and request/response format.
- [CLI reference](docs/CLI.md) – Commands and flags.
- [Development guide](docs/DEVELOPMENT.md) – Build, test, and release.

## Performance (guideline)

| Documents | Query time (approx) | Memory (approx) |
| --------- | ------------------- | --------------- |
| 10k       | 10–20 ms            | ~100 MB         |
| 100k      | 20–50 ms            | ~500 MB         |

Exact numbers depend on hardware, index type (in-memory vs FAISS), and embedder (production ONNX vs mock).

## License

MIT – see [LICENSE](LICENSE).
