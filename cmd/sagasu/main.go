// Package main is the Sagasu CLI entry point.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hyperjump/sagasu/internal/cli"
	"github.com/hyperjump/sagasu/internal/config"
	"github.com/hyperjump/sagasu/internal/embedding"
	"github.com/hyperjump/sagasu/internal/extract"
	"github.com/hyperjump/sagasu/internal/fileid"
	"github.com/hyperjump/sagasu/internal/indexer"
	"github.com/hyperjump/sagasu/internal/keyword"
	"github.com/hyperjump/sagasu/internal/models"
	"github.com/hyperjump/sagasu/internal/search"
	"github.com/hyperjump/sagasu/internal/server"
	"github.com/hyperjump/sagasu/internal/storage"
	"github.com/hyperjump/sagasu/internal/vector"
	"github.com/hyperjump/sagasu/internal/watcher"
	"go.uber.org/zap"
)

var version = "dev"

const defaultConfigPath = "/usr/local/etc/sagasu/config.yaml"

// loadConfig loads config from path. If path is the default and the file does not exist,
// it tries config.yaml in the current directory (for development).
// Returns the config and the path that was actually loaded (for saving, etc.).
func loadConfig(path string) (*config.Config, string, error) {
	cfg, err := config.Load(path)
	if err != nil {
		if path == defaultConfigPath {
			if unwrap := errors.Unwrap(err); unwrap != nil && os.IsNotExist(unwrap) {
				if cwd, cwdErr := os.Getwd(); cwdErr == nil {
					fallback := filepath.Join(cwd, "config.yaml")
					if _, statErr := os.Stat(fallback); statErr == nil {
						cfg, loadErr := config.Load(fallback)
						if loadErr != nil {
							return nil, "", loadErr
						}
						return cfg, fallback, nil
					}
				}
			}
		}
		return nil, "", err
	}
	return cfg, path, nil
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	command := os.Args[1]
	switch command {
	case "server":
		runServer()
	case "search":
		runSearch()
	case "index":
		runIndex()
	case "delete":
		runDelete()
	case "watch":
		runWatch()
	case "version", "--version", "-v":
		fmt.Printf("sagasu version %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func runServer() {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	configPath := fs.String("config", defaultConfigPath, "config file path")
	_ = fs.Parse(os.Args[2:])

	cfg, resolvedConfigPath, err := loadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	components, err := initializeComponents(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to initialize components", zap.Error(err))
	}
	defer components.Close()

	idx := components.Indexer
	exts := cfg.Watch.Extensions
	watchSvc := watcher.NewWatcher(
		cfg.Watch.Directories,
		exts,
		cfg.Watch.RecursiveOrDefault(),
		func(path string) {
			if err := idx.IndexFile(context.Background(), path, exts); err != nil {
				logger.Warn("watch index file failed", zap.String("path", path), zap.Error(err))
			}
		},
		func(path string) {
			if err := idx.DeleteDocument(context.Background(), fileid.FileDocID(path)); err != nil {
				logger.Warn("watch delete by path failed", zap.String("path", path), zap.Error(err))
			}
		},
	)
	watchCtx, watchCancel := context.WithCancel(context.Background())
	defer watchCancel()
	if err := watchSvc.Start(watchCtx); err != nil {
		logger.Fatal("Failed to start watcher", zap.Error(err))
	}
	watchSvc.SyncExistingFiles()

	srv := server.NewServer(
		components.Engine,
		components.Indexer,
		components.Storage,
		&cfg.Server,
		logger,
		watchSvc,
		resolvedConfigPath,
		cfg,
	)
	go func() {
		if err := srv.Start(); err != nil {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down...")
	watchCancel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Stop(ctx)
}

// searchArgsReorder moves any flags (and their values) that appear after the query
// to the front of the slice so that flag.Parse() sees them. Go's flag package
// stops at the first non-flag argument, so "sagasu search \"query\" -min-score 0.5"
// would otherwise leave -min-score unparsed (default 0.05 used).
func searchArgsReorder(args []string) []string {
	for i, a := range args {
		if len(a) > 0 && a[0] == '-' {
			if i == 0 {
				return args
			}
			reordered := make([]string, 0, len(args))
			reordered = append(reordered, args[i:]...)
			reordered = append(reordered, args[:i]...)
			return reordered
		}
	}
	return args
}

func runSearch() {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	configPath := fs.String("config", defaultConfigPath, "config file path")
	serverURL := fs.String("server", "http://localhost:8080", "server URL (empty = use direct storage)")
	limit := fs.Int("limit", 10, "number of results")
	minScore := fs.Float64("min-score", 0.05, "minimum score threshold (exclude results below)")
	kwWeight := fs.Float64("keyword-weight", 0.5, "keyword weight")
	semWeight := fs.Float64("semantic-weight", 0.5, "semantic weight")
	searchArgs := searchArgsReorder(os.Args[2:])
	_ = fs.Parse(searchArgs)

	if fs.NArg() < 1 {
		fmt.Println("Usage: sagasu search [flags] <query>")
		os.Exit(1)
	}
	queryStr := fs.Arg(0)

	searchQuery := &models.SearchQuery{
		Query:          queryStr,
		Limit:          *limit,
		MinScore:       *minScore,
		KeywordWeight:  *kwWeight,
		SemanticWeight: *semWeight,
	}

	if *serverURL != "" {
		// Use HTTP API when server is running (avoids Bleve/SQLite lock conflict).
		response, err := searchViaHTTP(*serverURL, searchQuery)
		if err != nil {
			fmt.Printf("Search failed: %v\n", err)
			os.Exit(1)
		}
		cli.PrintSearchResults(response)
		return
	}

	// Direct storage access (when server is not running).
	cfg, _, err := loadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	components, err := initializeComponents(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to initialize", zap.Error(err))
	}
	defer components.Close()

	response, err := components.Engine.Search(context.Background(), searchQuery)
	if err != nil {
		fmt.Printf("Search failed: %v\n", err)
		os.Exit(1)
	}
	cli.PrintSearchResults(response)
}

func searchViaHTTP(serverURL string, query *models.SearchQuery) (*models.SearchResponse, error) {
	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(serverURL+"/api/v1/search", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(b))
	}
	var response models.SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &response, nil
}

func runIndex() {
	fs := flag.NewFlagSet("index", flag.ExitOnError)
	configPath := fs.String("config", defaultConfigPath, "config file path")
	_ = fs.String("title", "", "document title (unused; document title is derived from filename)")
	_ = fs.Parse(os.Args[2:])

	if fs.NArg() < 1 {
		fmt.Println("Usage: sagasu index [flags] <file>")
		os.Exit(1)
	}
	filePath := fs.Arg(0)

	cfg, _, err := loadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	components, err := initializeComponents(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to initialize", zap.Error(err))
	}
	defer components.Close()

	// Use nil for allowedExts to accept any file (CLI index has no extension filter)
	if err := components.Indexer.IndexFile(context.Background(), filePath, nil); err != nil {
		fmt.Printf("Indexing failed: %v\n", err)
		os.Exit(1)
	}
	absPath, _ := filepath.Abs(filePath)
	docID := fileid.FileDocID(absPath)
	fmt.Printf("Document indexed successfully: %s\n", docID)
}

func runWatch() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: sagasu watch <add|remove|list> [path]")
		fmt.Println("  sagasu watch add <path>     Add directory to watch")
		fmt.Println("  sagasu watch remove <path>  Remove directory from watch")
		fmt.Println("  sagasu watch list           List watched directories")
		os.Exit(1)
	}
	sub := os.Args[2]
	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	serverURL := fs.String("server", "http://localhost:8080", "server URL")
	_ = fs.Parse(os.Args[3:])
	switch sub {
	case "add":
		if fs.NArg() < 1 {
			fmt.Println("Usage: sagasu watch add <path>")
			os.Exit(1)
		}
		path, _ := filepath.Abs(fs.Arg(0))
		body, _ := json.Marshal(map[string]interface{}{"path": path, "sync": true})
		resp, err := http.Post(*serverURL+"/api/v1/watch/directories", "application/json", bytes.NewReader(body))
		if err != nil {
			fmt.Printf("Request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			b, _ := io.ReadAll(resp.Body)
			fmt.Printf("Add failed (%d): %s\n", resp.StatusCode, string(b))
			os.Exit(1)
		}
		fmt.Printf("Added: %s\n", path)
	case "remove":
		if fs.NArg() < 1 {
			fmt.Println("Usage: sagasu watch remove <path>")
			os.Exit(1)
		}
		path, _ := filepath.Abs(fs.Arg(0))
		req, _ := http.NewRequest(http.MethodDelete, *serverURL+"/api/v1/watch/directories?path="+url.QueryEscape(path), nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("Request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			fmt.Printf("Remove failed (%d): %s\n", resp.StatusCode, string(b))
			os.Exit(1)
		}
		fmt.Printf("Removed: %s\n", path)
	case "list":
		resp, err := http.Get(*serverURL + "/api/v1/watch/directories")
		if err != nil {
			fmt.Printf("Request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			fmt.Printf("List failed (%d): %s\n", resp.StatusCode, string(b))
			os.Exit(1)
		}
		var out struct {
			Directories []string `json:"directories"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			fmt.Printf("Parse failed: %v\n", err)
			os.Exit(1)
		}
		for _, d := range out.Directories {
			fmt.Println(d)
		}
	default:
		fmt.Printf("Unknown watch subcommand: %s\n", sub)
		os.Exit(1)
	}
}

func runDelete() {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	configPath := fs.String("config", defaultConfigPath, "config file path")
	_ = fs.Parse(os.Args[2:])

	if fs.NArg() < 1 {
		fmt.Println("Usage: sagasu delete [flags] <document-id>")
		os.Exit(1)
	}
	docID := fs.Arg(0)

	cfg, _, err := loadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	components, err := initializeComponents(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to initialize", zap.Error(err))
	}
	defer components.Close()

	if err := components.Indexer.DeleteDocument(context.Background(), docID); err != nil {
		fmt.Printf("Deletion failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Document deleted: %s\n", docID)
}

// Components holds initialized services.
type Components struct {
	Storage      storage.Storage
	Embedder     embedding.Embedder
	VectorIndex  vector.VectorIndex
	KeywordIndex keyword.KeywordIndex
	Engine       *search.Engine
	Indexer      *indexer.Indexer
}

func (c *Components) Close() {
	if c.Storage != nil {
		_ = c.Storage.Close()
	}
	if c.Embedder != nil {
		_ = c.Embedder.Close()
	}
	if c.VectorIndex != nil {
		_ = c.VectorIndex.Close()
	}
	if c.KeywordIndex != nil {
		_ = c.KeywordIndex.Close()
	}
}

func initializeComponents(cfg *config.Config, logger *zap.Logger) (*Components, error) {
	store, err := storage.NewSQLiteStorage(cfg.Storage.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	var embedder embedding.Embedder
	onnxEmbedder, err := embedding.NewONNXEmbedder(
		cfg.Embedding.ModelPath,
		cfg.Embedding.Dimensions,
		cfg.Embedding.MaxTokens,
		cfg.Embedding.CacheSize,
	)
	if err != nil {
		embedder = embedding.NewMockEmbedder(cfg.Embedding.Dimensions)
	} else {
		embedder = onnxEmbedder
	}

	vectorIndex, err := vector.NewMemoryIndex(cfg.Embedding.Dimensions)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize vector index: %w", err)
	}

	keywordIndex, err := keyword.NewBleveIndex(cfg.Storage.BleveIndexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize keyword index: %w", err)
	}

	engine := search.NewEngine(store, embedder, vectorIndex, keywordIndex, &cfg.Search)
	idx := indexer.NewIndexer(store, embedder, vectorIndex, keywordIndex, &cfg.Search, extract.NewExtractor())

	return &Components{
		Storage:      store,
		Embedder:     embedder,
		VectorIndex:  vectorIndex,
		KeywordIndex: keywordIndex,
		Engine:       engine,
		Indexer:      idx,
	}, nil
}

func printUsage() {
	fmt.Println(`sagasu - Fast local hybrid search engine

Usage:
  sagasu server [flags]           Start the HTTP server
  sagasu search [flags] <query>   Search documents
  sagasu index [flags] <file>     Index a document
  sagasu delete [flags] <id>       Delete a document
  sagasu watch <add|remove|list>  Manage watched directories
  sagasu version                  Show version
  sagasu help                     Show this help

Server Flags:
  --config string    Config file path (default: /usr/local/etc/sagasu/config.yaml)

Search Flags:
  --config string           Config file path (for direct storage mode)
  --server string           Server URL (default: http://localhost:8080). Use empty to access storage directly.
  --limit int               Number of results (default: 10)
  --min-score float         Minimum score threshold (default: 0.05)
  --keyword-weight float    Keyword weight (default: 0.5)
  --semantic-weight float   Semantic weight (default: 0.5)

Index Flags:
  --config string    Config file path
  --title string     Document title

Watch Flags:
  --server string    Server URL (default: http://localhost:8080)

Examples:
  sagasu server
  sagasu search "machine learning algorithms"
  sagasu search --min-score 0.1 "raosan"
  sagasu search --keyword-weight 0.7 --semantic-weight 0.3 "neural networks"
  sagasu index --title "My Document" document.txt
  sagasu delete doc-123
  sagasu watch add /path/to/docs
  sagasu watch list`)
}
