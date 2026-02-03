package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hyperjump/sagasu/internal/models"
	"go.uber.org/zap"
)

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var query models.SearchQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
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

func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{"error": message})
}
