// Package server provides the HTTP API for Sagasu.
package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hyperjump/sagasu/internal/config"
	"github.com/hyperjump/sagasu/internal/indexer"
	"github.com/hyperjump/sagasu/internal/search"
	"github.com/hyperjump/sagasu/internal/storage"
	"github.com/hyperjump/sagasu/internal/watcher"
	"go.uber.org/zap"
)

// WatchDirectoryService provides list/add/remove of watched directories (optional).
type WatchDirectoryService interface {
	Directories() []string
	AddDirectory(path string, syncExisting bool) error
	RemoveDirectory(path string) error
}

// Server is the HTTP server for the Sagasu API.
type Server struct {
	engine       *search.Engine
	indexer      *indexer.Indexer
	storage      storage.Storage
	config       *config.ServerConfig
	logger       *zap.Logger
	server       *http.Server
	watch        WatchDirectoryService
	configPath   string
	watchConfig  *config.Config
	watchConfigMu sync.Mutex
}

// NewServer creates a server with the given dependencies.
// watchSvc is optional; if non-nil, watch directory endpoints are enabled.
// configPath and fullCfg are optional; if both set, add/remove directory persists to config file.
func NewServer(
	engine *search.Engine,
	idx *indexer.Indexer,
	storage storage.Storage,
	cfg *config.ServerConfig,
	logger *zap.Logger,
	watchSvc WatchDirectoryService,
	configPath string,
	fullCfg *config.Config,
) *Server {
	return &Server{
		engine:      engine,
		indexer:     idx,
		storage:     storage,
		config:      cfg,
		logger:      logger,
		watch:       watchSvc,
		configPath:  configPath,
		watchConfig: fullCfg,
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
	r.Get("/api/v1/watch/directories", s.handleWatchDirectoriesList)
	r.Post("/api/v1/watch/directories", s.handleWatchDirectoriesAdd)
	r.Delete("/api/v1/watch/directories", s.handleWatchDirectoriesRemove)
	r.Get("/health", s.handleHealth)

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: r,
	}
	s.logger.Info("Starting server", zap.String("addr", addr))
	return s.server.ListenAndServe()
}

// Stop gracefully shuts down the server. If the server was created with a watcher
// that implements the Stop method (e.g. *watcher.Watcher), it is stopped first.
func (s *Server) Stop(ctx context.Context) error {
	if w, ok := s.watch.(*watcher.Watcher); ok && w != nil {
		w.Stop()
	}
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}
