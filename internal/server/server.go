// Package server provides the HTTP API for Sagasu.
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hyperjump/sagasu/internal/config"
	"github.com/hyperjump/sagasu/internal/indexer"
	"github.com/hyperjump/sagasu/internal/search"
	"github.com/hyperjump/sagasu/internal/storage"
	"go.uber.org/zap"
)

// Server is the HTTP server for the Sagasu API.
type Server struct {
	engine  *search.Engine
	indexer *indexer.Indexer
	storage storage.Storage
	config  *config.ServerConfig
	logger  *zap.Logger
	server  *http.Server
}

// NewServer creates a server with the given dependencies.
func NewServer(
	engine *search.Engine,
	idx *indexer.Indexer,
	storage storage.Storage,
	cfg *config.ServerConfig,
	logger *zap.Logger,
) *Server {
	return &Server{
		engine:  engine,
		indexer: idx,
		storage: storage,
		config:  cfg,
		logger:  logger,
	}
}

// Start starts the HTTP server and blocks until it stops.
func (s *Server) Start() error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.Compress(5))

	r.Post("/api/v1/search", s.handleSearch)
	r.Post("/api/v1/documents", s.handleIndexDocument)
	r.Get("/api/v1/documents/{id}", s.handleGetDocument)
	r.Delete("/api/v1/documents/{id}", s.handleDeleteDocument)
	r.Get("/health", s.handleHealth)

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: r,
	}
	s.logger.Info("Starting server", zap.String("addr", addr))
	return s.server.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}
