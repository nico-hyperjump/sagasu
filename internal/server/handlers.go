package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/hyperjump/sagasu/internal/config"
	"github.com/hyperjump/sagasu/internal/models"
	"github.com/hyperjump/sagasu/internal/storage"
	"go.uber.org/zap"
)

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var query models.SearchQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	s.logger.Debug("search request", zap.String("query", query.Query), zap.Int("limit", query.Limit))
	response, err := s.engine.Search(r.Context(), &query)
	if err != nil {
		s.logger.Error("search failed", zap.Error(err))
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleIndexDocument(w http.ResponseWriter, r *http.Request) {
	var input models.DocumentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	s.logger.Debug("index document request", zap.String("id", input.ID), zap.String("title", input.Title))
	if err := s.indexer.IndexDocument(r.Context(), &input); err != nil {
		s.logger.Error("indexing failed", zap.Error(err))
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.respondJSON(w, http.StatusCreated, map[string]string{"id": input.ID, "status": "indexed"})
}

func (s *Server) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	doc, err := s.storage.GetDocument(r.Context(), id)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "document not found")
		return
	}
	s.respondJSON(w, http.StatusOK, doc)
}

func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s.logger.Debug("delete document request", zap.String("id", id))
	if err := s.indexer.DeleteDocument(r.Context(), id); err != nil {
		s.logger.Error("deletion failed", zap.Error(err))
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	docCount, err := s.storage.CountDocuments(ctx)
	if err != nil {
		s.logger.Error("status: count documents failed", zap.Error(err))
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	chunkCount, err := s.storage.CountChunks(ctx)
	if err != nil {
		s.logger.Error("status: count chunks failed", zap.Error(err))
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	vectorSize := s.engine.VectorIndexSize()
	vectorIndexType := s.engine.VectorIndexType()
	resp := map[string]interface{}{
		"documents":         docCount,
		"chunks":            chunkCount,
		"vector_index_size": vectorSize,
	}

	// Add configuration info
	configInfo := map[string]interface{}{
		"vector_index_type": vectorIndexType,
	}
	if s.watchConfig != nil {
		configInfo["embedding_dimensions"] = s.watchConfig.Embedding.Dimensions
		configInfo["chunk_size"] = s.watchConfig.Search.ChunkSize
		configInfo["chunk_overlap"] = s.watchConfig.Search.ChunkOverlap
		configInfo["ranking_enabled"] = s.watchConfig.Search.RankingEnabled
		configInfo["database_path"] = s.watchConfig.Storage.DatabasePath
		configInfo["bleve_index_path"] = s.watchConfig.Storage.BleveIndexPath
		configInfo["faiss_index_path"] = s.watchConfig.Storage.FAISSIndexPath

		diskBytes, err := storage.DiskUsageBytes(
			s.watchConfig.Storage.DatabasePath,
			s.watchConfig.Storage.BleveIndexPath,
			s.watchConfig.Storage.FAISSIndexPath,
		)
		if err == nil {
			resp["disk_usage_bytes"] = diskBytes
		}
	}
	resp["config"] = configInfo
	s.respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleWatchDirectoriesList(w http.ResponseWriter, r *http.Request) {
	if s.watch == nil {
		s.respondError(w, http.StatusNotImplemented, "watch not enabled")
		return
	}
	dirs := s.watch.Directories()
	s.respondJSON(w, http.StatusOK, map[string]interface{}{"directories": dirs})
}

type watchAddRequest struct {
	Path string `json:"path"`
	Sync *bool  `json:"sync,omitempty"`
}

func (s *Server) handleWatchDirectoriesAdd(w http.ResponseWriter, r *http.Request) {
	if s.watch == nil {
		s.respondError(w, http.StatusNotImplemented, "watch not enabled")
		return
	}
	var req watchAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Path == "" {
		s.respondError(w, http.StatusBadRequest, "path is required")
		return
	}
	abs, err := filepath.Abs(req.Path)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid path")
		return
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			s.respondError(w, http.StatusNotFound, "directory not found")
			return
		}
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !info.IsDir() {
		s.respondError(w, http.StatusBadRequest, "path is not a directory")
		return
	}
	syncExisting := true
	if req.Sync != nil {
		syncExisting = *req.Sync
	}
	s.logger.Debug("watch add directory request", zap.String("path", abs), zap.Bool("sync_existing", syncExisting))
	if err := s.watch.AddDirectory(abs, syncExisting); err != nil {
		s.logger.Error("watch add directory failed", zap.Error(err))
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if s.configPath != "" && s.watchConfig != nil {
		s.watchConfigMu.Lock()
		s.watchConfig.Watch.Directories = s.watch.Directories()
		err := config.Save(s.configPath, s.watchConfig)
		s.watchConfigMu.Unlock()
		if err != nil {
			s.logger.Warn("failed to persist watch config", zap.Error(err))
		}
	}
	s.respondJSON(w, http.StatusCreated, map[string]string{"path": abs, "status": "added"})
}

func (s *Server) handleWatchDirectoriesRemove(w http.ResponseWriter, r *http.Request) {
	if s.watch == nil {
		s.respondError(w, http.StatusNotImplemented, "watch not enabled")
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		var body struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil && body.Path != "" {
			path = body.Path
		}
	}
	if path == "" {
		s.respondError(w, http.StatusBadRequest, "path is required (query or body)")
		return
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid path")
		return
	}
	s.logger.Debug("watch remove directory request", zap.String("path", abs))
	if err := s.watch.RemoveDirectory(abs); err != nil {
		s.logger.Error("watch remove directory failed", zap.Error(err))
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if s.configPath != "" && s.watchConfig != nil {
		s.watchConfigMu.Lock()
		s.watchConfig.Watch.Directories = s.watch.Directories()
		err := config.Save(s.configPath, s.watchConfig)
		s.watchConfigMu.Unlock()
		if err != nil {
			s.logger.Warn("failed to persist watch config", zap.Error(err))
		}
	}
	s.respondJSON(w, http.StatusOK, map[string]string{"path": abs, "status": "removed"})
}

func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{"error": message})
}
