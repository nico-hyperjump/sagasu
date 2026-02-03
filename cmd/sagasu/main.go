// Package main is the Sagasu CLI entry point.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hyperjump/sagasu/internal/cli"
	"github.com/hyperjump/sagasu/internal/config"
	"github.com/hyperjump/sagasu/internal/embedding"
	"github.com/hyperjump/sagasu/internal/indexer"
	"github.com/hyperjump/sagasu/internal/keyword"
	"github.com/hyperjump/sagasu/internal/models"
	"github.com/hyperjump/sagasu/internal/search"
	"github.com/hyperjump/sagasu/internal/server"
	"github.com/hyperjump/sagasu/internal/storage"
	"github.com/hyperjump/sagasu/internal/vector"
	"go.uber.org/zap"
)

var version = "dev"

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
	configPath := fs.String("config", "/usr/local/etc/sagasu/config.yaml", "config file path")
	_ = fs.Parse(os.Args[2:])

	cfg, err := config.Load(*configPath)
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

	srv := server.NewServer(
		components.Engine,
		components.Indexer,
		components.Storage,
		&cfg.Server,
		logger,
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Stop(ctx)
}

func runSearch() {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	configPath := fs.String("config", "/usr/local/etc/sagasu/config.yaml", "config file path")
	limit := fs.Int("limit", 10, "number of results")
	kwWeight := fs.Float64("keyword-weight", 0.5, "keyword weight")
	semWeight := fs.Float64("semantic-weight", 0.5, "semantic weight")
	_ = fs.Parse(os.Args[2:])

	if fs.NArg() < 1 {
		fmt.Println("Usage: sagasu search [flags] <query>")
		os.Exit(1)
	}
	queryStr := fs.Arg(0)

	cfg, err := config.Load(*configPath)
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

	searchQuery := &models.SearchQuery{
		Query:          queryStr,
		Limit:          *limit,
		KeywordWeight:  *kwWeight,
		SemanticWeight: *semWeight,
	}
	response, err := components.Engine.Search(context.Background(), searchQuery)
	if err != nil {
		fmt.Printf("Search failed: %v\n", err)
		os.Exit(1)
	}
	cli.PrintSearchResults(response)
}

func runIndex() {
	fs := flag.NewFlagSet("index", flag.ExitOnError)
	configPath := fs.String("config", "/usr/local/etc/sagasu/config.yaml", "config file path")
	title := fs.String("title", "", "document title")
	_ = fs.Parse(os.Args[2:])

	if fs.NArg() < 1 {
		fmt.Println("Usage: sagasu index [flags] <file>")
		os.Exit(1)
	}
	filePath := fs.Arg(0)

	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Failed to read file: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load(*configPath)
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

	input := &models.DocumentInput{
		Title:   *title,
		Content: string(content),
	}
	if err := components.Indexer.IndexDocument(context.Background(), input); err != nil {
		fmt.Printf("Indexing failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Document indexed successfully: %s\n", input.ID)
}

func runDelete() {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	configPath := fs.String("config", "/usr/local/etc/sagasu/config.yaml", "config file path")
	_ = fs.Parse(os.Args[2:])

	if fs.NArg() < 1 {
		fmt.Println("Usage: sagasu delete [flags] <document-id>")
		os.Exit(1)
	}
	docID := fs.Arg(0)

	cfg, err := config.Load(*configPath)
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
	idx := indexer.NewIndexer(store, embedder, vectorIndex, keywordIndex, &cfg.Search)

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
  sagasu delete [flags] <id>      Delete a document
  sagasu version                  Show version
  sagasu help                     Show this help

Server Flags:
  --config string    Config file path (default: /usr/local/etc/sagasu/config.yaml)

Search Flags:
  --config string           Config file path
  --limit int               Number of results (default: 10)
  --keyword-weight float    Keyword weight (default: 0.5)
  --semantic-weight float   Semantic weight (default: 0.5)

Index Flags:
  --config string    Config file path
  --title string     Document title

Examples:
  sagasu server
  sagasu search "machine learning algorithms"
  sagasu search --keyword-weight 0.7 --semantic-weight 0.3 "neural networks"
  sagasu index --title "My Document" document.txt
  sagasu delete doc-123
`)
}
