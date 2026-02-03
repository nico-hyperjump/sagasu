package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hyperjump/sagasu/internal/config"
	"github.com/hyperjump/sagasu/internal/embedding"
	"github.com/hyperjump/sagasu/internal/indexer"
	"github.com/hyperjump/sagasu/internal/keyword"
	"github.com/hyperjump/sagasu/internal/models"
	"github.com/hyperjump/sagasu/internal/search"
	"github.com/hyperjump/sagasu/internal/storage"
	"github.com/hyperjump/sagasu/internal/vector"
	"go.uber.org/zap"
)

type mockWatchService struct {
	dirs []string
}

func (m *mockWatchService) Directories() []string {
	return append([]string(nil), m.dirs...)
}

func (m *mockWatchService) AddDirectory(path string, _ bool) error {
	for _, d := range m.dirs {
		if d == path {
			return nil
		}
	}
	m.dirs = append(m.dirs, path)
	return nil
}

func (m *mockWatchService) RemoveDirectory(path string) error {
	for i, d := range m.dirs {
		if d == path {
			m.dirs = append(m.dirs[:i], m.dirs[i+1:]...)
			return nil
		}
	}
	return nil
}

func TestHandleWatchDirectoriesList(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewSQLiteStorage(dir + "/db.sqlite")
	defer store.Close()
	embedder := embedding.NewMockEmbedder(4)
	defer embedder.Close()
	vecIdx, _ := vector.NewMemoryIndex(4)
	defer vecIdx.Close()
	kwIdx, _ := keyword.NewBleveIndex(dir + "/bleve")
	defer kwIdx.Close()
	cfg := &config.SearchConfig{ChunkSize: 10, ChunkOverlap: 2, TopKCandidates: 20,
		DefaultKeywordWeight: 0.5, DefaultSemanticWeight: 0.5}
	engine := search.NewEngine(store, embedder, vecIdx, kwIdx, cfg)
	idx := indexer.NewIndexer(store, embedder, vecIdx, kwIdx, cfg, nil)
	logger := zap.NewNop()

	mock := &mockWatchService{dirs: []string{"/tmp/docs"}}
	srv := NewServer(engine, idx, store, &config.ServerConfig{Port: 8080}, logger, mock, "", nil)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/watch/directories", nil)
	w := httptest.NewRecorder()
	srv.handleWatchDirectoriesList(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d", w.Code)
	}
	var out struct {
		Directories []string `json:"directories"`
	}
	if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Directories) != 1 || out.Directories[0] != "/tmp/docs" {
		t.Errorf("directories: got %v", out.Directories)
	}
}

func TestHandleWatchDirectoriesList_NotEnabled(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewSQLiteStorage(dir + "/db.sqlite")
	defer store.Close()
	embedder := embedding.NewMockEmbedder(4)
	defer embedder.Close()
	vecIdx, _ := vector.NewMemoryIndex(4)
	defer vecIdx.Close()
	kwIdx, _ := keyword.NewBleveIndex(dir + "/bleve")
	defer kwIdx.Close()
	cfg := &config.SearchConfig{ChunkSize: 10, ChunkOverlap: 2, TopKCandidates: 20,
		DefaultKeywordWeight: 0.5, DefaultSemanticWeight: 0.5}
	engine := search.NewEngine(store, embedder, vecIdx, kwIdx, cfg)
	idx := indexer.NewIndexer(store, embedder, vecIdx, kwIdx, cfg, nil)
	logger := zap.NewNop()

	srv := NewServer(engine, idx, store, &config.ServerConfig{Port: 8080}, logger, nil, "", nil)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/watch/directories", nil)
	w := httptest.NewRecorder()
	srv.handleWatchDirectoriesList(w, r)
	if w.Code != http.StatusNotImplemented {
		t.Errorf("status: got %d, want 501", w.Code)
	}
}

func TestHandleWatchDirectoriesAdd(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewSQLiteStorage(dir + "/db.sqlite")
	defer store.Close()
	embedder := embedding.NewMockEmbedder(4)
	defer embedder.Close()
	vecIdx, _ := vector.NewMemoryIndex(4)
	defer vecIdx.Close()
	kwIdx, _ := keyword.NewBleveIndex(dir + "/bleve")
	defer kwIdx.Close()
	cfg := &config.SearchConfig{ChunkSize: 10, ChunkOverlap: 2, TopKCandidates: 20,
		DefaultKeywordWeight: 0.5, DefaultSemanticWeight: 0.5}
	engine := search.NewEngine(store, embedder, vecIdx, kwIdx, cfg)
	idx := indexer.NewIndexer(store, embedder, vecIdx, kwIdx, cfg, nil)
	logger := zap.NewNop()

	mock := &mockWatchService{}
	srv := NewServer(engine, idx, store, &config.ServerConfig{Port: 8080}, logger, mock, "", nil)

	body, _ := json.Marshal(map[string]string{"path": dir})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/watch/directories", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleWatchDirectoriesAdd(w, r)
	if w.Code != http.StatusCreated {
		t.Errorf("status: got %d, body: %s", w.Code, w.Body.String())
	}
	if len(mock.Directories()) != 1 {
		t.Errorf("expected 1 directory, got %v", mock.Directories())
	}
}

func TestHandleWatchDirectoriesAdd_InvalidPath(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewSQLiteStorage(dir + "/db.sqlite")
	defer store.Close()
	embedder := embedding.NewMockEmbedder(4)
	defer embedder.Close()
	vecIdx, _ := vector.NewMemoryIndex(4)
	defer vecIdx.Close()
	kwIdx, _ := keyword.NewBleveIndex(dir + "/bleve")
	defer kwIdx.Close()
	cfg := &config.SearchConfig{ChunkSize: 10, ChunkOverlap: 2, TopKCandidates: 20,
		DefaultKeywordWeight: 0.5, DefaultSemanticWeight: 0.5}
	engine := search.NewEngine(store, embedder, vecIdx, kwIdx, cfg)
	idx := indexer.NewIndexer(store, embedder, vecIdx, kwIdx, cfg, nil)
	logger := zap.NewNop()

	mock := &mockWatchService{}
	srv := NewServer(engine, idx, store, &config.ServerConfig{Port: 8080}, logger, mock, "", nil)

	body, _ := json.Marshal(map[string]string{"path": dir + "/nonexistent"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/watch/directories", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleWatchDirectoriesAdd(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d", w.Code)
	}
}

func TestHandleWatchDirectoriesRemove(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewSQLiteStorage(dir + "/db.sqlite")
	defer store.Close()
	embedder := embedding.NewMockEmbedder(4)
	defer embedder.Close()
	vecIdx, _ := vector.NewMemoryIndex(4)
	defer vecIdx.Close()
	kwIdx, _ := keyword.NewBleveIndex(dir + "/bleve")
	defer kwIdx.Close()
	cfg := &config.SearchConfig{ChunkSize: 10, ChunkOverlap: 2, TopKCandidates: 20,
		DefaultKeywordWeight: 0.5, DefaultSemanticWeight: 0.5}
	engine := search.NewEngine(store, embedder, vecIdx, kwIdx, cfg)
	idx := indexer.NewIndexer(store, embedder, vecIdx, kwIdx, cfg, nil)
	logger := zap.NewNop()

	mock := &mockWatchService{dirs: []string{dir}}
	srv := NewServer(engine, idx, store, &config.ServerConfig{Port: 8080}, logger, mock, "", nil)

	r := httptest.NewRequest(http.MethodDelete, "/api/v1/watch/directories?path="+dir, nil)
	w := httptest.NewRecorder()
	srv.handleWatchDirectoriesRemove(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d", w.Code)
	}
	if len(mock.Directories()) != 0 {
		t.Errorf("expected 0 directories, got %v", mock.Directories())
	}
}

func TestHandleSearch(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewSQLiteStorage(dir + "/db.sqlite")
	defer store.Close()
	embedder := embedding.NewMockEmbedder(4)
	defer embedder.Close()
	vecIdx, _ := vector.NewMemoryIndex(4)
	defer vecIdx.Close()
	kwIdx, _ := keyword.NewBleveIndex(dir + "/bleve")
	defer kwIdx.Close()
	cfg := &config.SearchConfig{ChunkSize: 10, ChunkOverlap: 2, TopKCandidates: 20,
		DefaultKeywordWeight: 0.5, DefaultSemanticWeight: 0.5}
	engine := search.NewEngine(store, embedder, vecIdx, kwIdx, cfg)
	idx := indexer.NewIndexer(store, embedder, vecIdx, kwIdx, cfg, nil)
	_ = idx.IndexDocument(context.Background(), &models.DocumentInput{ID: "d1", Title: "T", Content: "hello world"})
	logger := zap.NewNop()

	srv := NewServer(engine, idx, store, &config.ServerConfig{Port: 8080}, logger, nil, "", nil)
	body, _ := json.Marshal(map[string]string{"query": "hello"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleSearch(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d", w.Code)
	}
}

func TestHandleStatus(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewSQLiteStorage(dir + "/db.sqlite")
	defer store.Close()
	embedder := embedding.NewMockEmbedder(4)
	defer embedder.Close()
	vecIdx, _ := vector.NewMemoryIndex(4)
	defer vecIdx.Close()
	kwIdx, _ := keyword.NewBleveIndex(dir + "/bleve")
	defer kwIdx.Close()
	cfg := &config.SearchConfig{ChunkSize: 10, ChunkOverlap: 2, TopKCandidates: 20,
		DefaultKeywordWeight: 0.5, DefaultSemanticWeight: 0.5}
	engine := search.NewEngine(store, embedder, vecIdx, kwIdx, cfg)
	idx := indexer.NewIndexer(store, embedder, vecIdx, kwIdx, cfg, nil)
	_ = idx.IndexDocument(context.Background(), &models.DocumentInput{ID: "d1", Title: "T", Content: "hello world"})
	logger := zap.NewNop()

	srv := NewServer(engine, idx, store, &config.ServerConfig{Port: 8080}, logger, nil, "", nil)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	srv.handleStatus(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, body: %s", w.Code, w.Body.String())
	}
	var out struct {
		Documents       int64 `json:"documents"`
		Chunks          int64 `json:"chunks"`
		VectorIndexSize int   `json:"vector_index_size"`
	}
	if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Documents != 1 {
		t.Errorf("documents: got %d, want 1", out.Documents)
	}
	if out.Chunks < 1 {
		t.Errorf("chunks: got %d, want >= 1", out.Chunks)
	}
	if out.VectorIndexSize < 1 {
		t.Errorf("vector_index_size: got %d, want >= 1", out.VectorIndexSize)
	}
}

func TestHandleStatus_WithDiskUsage(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewSQLiteStorage(dir + "/db.sqlite")
	defer store.Close()
	embedder := embedding.NewMockEmbedder(4)
	defer embedder.Close()
	vecIdx, _ := vector.NewMemoryIndex(4)
	defer vecIdx.Close()
	kwIdx, _ := keyword.NewBleveIndex(dir + "/bleve")
	defer kwIdx.Close()
	cfg := &config.SearchConfig{ChunkSize: 10, ChunkOverlap: 2, TopKCandidates: 20,
		DefaultKeywordWeight: 0.5, DefaultSemanticWeight: 0.5}
	engine := search.NewEngine(store, embedder, vecIdx, kwIdx, cfg)
	idx := indexer.NewIndexer(store, embedder, vecIdx, kwIdx, cfg, nil)
	_ = idx.IndexDocument(context.Background(), &models.DocumentInput{ID: "d1", Title: "T", Content: "hello world"})
	logger := zap.NewNop()

	fullCfg := &config.Config{
		Storage: config.StorageConfig{
			DatabasePath:   dir + "/db.sqlite",
			BleveIndexPath: dir + "/bleve",
			FAISSIndexPath: dir + "/faiss",
		},
	}
	srv := NewServer(engine, idx, store, &config.ServerConfig{Port: 8080}, logger, nil, "", fullCfg)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	srv.handleStatus(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, body: %s", w.Code, w.Body.String())
	}
	var out struct {
		Documents       int64  `json:"documents"`
		Chunks          int64  `json:"chunks"`
		VectorIndexSize int    `json:"vector_index_size"`
		DiskUsageBytes  *int64 `json:"disk_usage_bytes"`
	}
	if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.DiskUsageBytes == nil {
		t.Error("expected disk_usage_bytes in response when watchConfig is set")
	}
	if out.DiskUsageBytes != nil && *out.DiskUsageBytes < 1 {
		t.Errorf("disk_usage_bytes: got %d, want >= 1", *out.DiskUsageBytes)
	}
}
