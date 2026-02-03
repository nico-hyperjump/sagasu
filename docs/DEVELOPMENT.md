# Sagasu Development Guide

## Prerequisites

- **Go 1.21 or later**
- **CGO enabled** (required for production build with real embeddings)
- **ONNX Runtime** – for production semantic search (real embedder). Without it, the app falls back to a mock embedder.

```bash
# macOS
brew install onnxruntime
```

Optional:

- **FAISS** – for production vector index at scale (currently the app uses an in-memory index by default)

## Building

### Production build (real embeddings)

Ensure ONNX Runtime is installed, then:

```bash
make build
```

This uses `CGO_ENABLED=1` and links to the ONNX Runtime library. You also need the embedding model on disk (see [Production install from source](#production-install-from-source) or the formula’s `post_install`).

### Development build (mock embedder)

For quick local testing without ONNX or the model:

```bash
CGO_ENABLED=0 go build -o bin/sagasu ./cmd/sagasu
```

The binary will use a mock embedder and in-memory vector index.

## Production install from source

End-to-end steps for a real production-style install (no mock):

1. **Install dependencies**

   ```bash
   brew install onnxruntime   # macOS
   ```

2. **Clone and build**

   ```bash
   git clone https://github.com/hyperjump/sagasu.git
   cd sagasu
   make build
   ```

3. **Download the embedding model** (one-time, ~80MB)

   ```bash
   ./scripts/download-model.sh /usr/local/var/sagasu/data/models
   ```

   Or use another directory and set `embedding.model_path` in config accordingly.

4. **Install binary and config**

   ```bash
   make install
   ```

   This copies the binary to `/usr/local/bin/sagasu`, installs `config.yaml` to `/usr/local/etc/sagasu/`, and creates data dirs under `/usr/local/var/sagasu/data/`.

5. **Run the server**
   ```bash
   sagasu server
   ```
   Or use a service manager (e.g. launchd, systemd) with `--config /usr/local/etc/sagasu/config.yaml`.

## Running tests

```bash
go test -v ./...
go test -race ./...
```

Run only unit tests (skip integration that need Bleve/SQLite on disk):

```bash
go test -v ./internal/...
```

Integration tests (under `test/integration/`) use real SQLite and Bleve and a temp directory.

## Benchmarks

```bash
go test -bench=. -benchmem ./test/benchmark/
```

## Project layout

- `cmd/sagasu/` – CLI entrypoint
- `internal/config/` – Configuration loading
- `internal/models/` – Document, query, result types
- `internal/storage/` – SQLite persistence
- `internal/embedding/` – Embedder interface, cache, tokenizer, ONNX (optional)
- `internal/vector/` – Vector index interface, in-memory implementation
- `internal/keyword/` – Bleve keyword index
- `internal/search/` – Fusion, processor, highlighter, engine
- `internal/indexer/` – Chunker, preprocessor, indexer
- `internal/server/` – HTTP server and handlers
- `internal/cli/` – CLI helpers
- `pkg/utils/` – Shared utilities
- `test/integration/` – Integration tests
- `test/benchmark/` – Benchmarks
- `test/testdata/` – Sample data
- `docs/` – API, CLI, and development docs

## Adding tests

- Prefer table-driven unit tests.
- Use interfaces and dependency injection so components can be tested with mocks (e.g. `embedding.MockEmbedder`, `vector.MemoryIndex`).
- Integration tests that need real storage/indices should use `t.TempDir()` and create SQLite/Bleve there.

## Releasing

1. Bump version and tag: `git tag -a v1.0.0 -m "Release v1.0.0" && git push origin v1.0.0`
2. Compute SHA256 for the tarball and update `Formula/sagasu.rb` (url, version, sha256).
3. Document in the repo how to run `make release` and update the Homebrew formula.
