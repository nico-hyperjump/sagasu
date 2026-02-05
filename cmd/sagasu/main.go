// Package main is the Sagasu CLI entry point.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
	"github.com/hyperjump/sagasu/pkg/utils"
	"go.uber.org/zap"
)

var version = "dev"

const defaultConfigPath = "/usr/local/etc/sagasu/config.yaml"

// loadConfig loads config from path. When path is the default, it first looks for
// config.yaml in the current directory (for development); if that exists it is used,
// so that "sagasu server" from the project dir uses the project's config (including debug).
// Returns the config and the path that was actually loaded (for saving, etc.).
func loadConfig(path string) (*config.Config, string, error) {
	if path == defaultConfigPath {
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
	cfg, err := config.Load(path)
	if err != nil {
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
	case "status":
		runStatus()
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
	debug := fs.Bool("debug", false, "enable debug logging (directory changes, file indexing, etc.)")
	_ = fs.Parse(os.Args[2:])

	cfg, resolvedConfigPath, err := loadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}
	debugMode := cfg.Debug || *debug
	logger, err := utils.NewLogger(debugMode)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("config loaded",
		zap.String("config_path", resolvedConfigPath),
		zap.Bool("debug", debugMode),
	)

	components, err := initializeComponents(cfg, logger, debugMode)
	if err != nil {
		logger.Fatal("Failed to initialize components", zap.Error(err))
	}
	defer components.Close()

	idx := components.Indexer
	exts := cfg.Watch.Extensions
	watchOpts := []watcher.WatcherOption{}
	if debugMode {
		watchOpts = append(watchOpts, watcher.WithLogger(logger))
	}
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
		watchOpts...,
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
	if cfg.Storage.FAISSIndexPath != "" && components.VectorIndex != nil {
		if err := components.VectorIndex.Save(cfg.Storage.FAISSIndexPath); err != nil && logger != nil {
			logger.Warn("vector index save failed", zap.String("path", cfg.Storage.FAISSIndexPath), zap.Error(err))
		}
	}
	watchCancel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Stop(ctx)
}

// printSearchUsage prints search subcommand usage and search efficiency hints.
func printSearchUsage(fs *flag.FlagSet) {
	fmt.Fprintf(fs.Output(), "Usage: sagasu search [flags] <query>\n\n")
	fmt.Fprintf(fs.Output(), "Query is all remaining arguments joined by spaces. Multi-word queries work with or without quotes.\n\n")
	fs.PrintDefaults()
	fmt.Fprintf(fs.Output(), `
Results are split into two lists: keyword matches and semantic-only matches.
  • Use --keyword=false for semantic-only search.
  • Use --semantic=false for keyword-only search.
  • Use --fuzzy to enable typo tolerance (finds results despite spelling mistakes).
  • --min-keyword-score and --min-semantic-score filter low-relevance hits; --limit controls how many per list.

Examples:
  sagasu search machine learning
  sagasu search "machine learning"                 # same as above
  sagasu search --keyword=false neural networks     # semantic-only
  sagasu search --fuzzy propodal                    # typo-tolerant search
  sagasu search --min-keyword-score 0.1 --min-semantic-score 0.2 --limit 20 your query
`)
}

// buildSearchQuery joins all positional args with spaces so multi-word queries
// work the same with or without shell quoting (e.g. "hyperjump profile" vs hyperjump profile).
func buildSearchQuery(args []string) string {
	return strings.TrimSpace(strings.Join(args, " "))
}

// searchConfigPathFromArgs returns the value of -config/--config from args if present, else defaultPath.
func searchConfigPathFromArgs(args []string, defaultPath string) string {
	for i, a := range args {
		if (a == "-config" || a == "--config") && i+1 < len(args) {
			return args[i+1]
		}
	}
	return defaultPath
}

// searchMinScoreDefaultsFromConfig loads config at path and returns default min keyword and semantic scores.
// On load failure, returns 0.49 for both. Zero values from config are accepted (meaning no filtering).
func searchMinScoreDefaultsFromConfig(path string) (minKeyword, minSemantic float64) {
	minKeyword, minSemantic = 0.49, 0.49
	cfg, _, err := loadConfig(path)
	if err != nil || cfg == nil {
		return minKeyword, minSemantic
	}
	// Use config values directly - 0 is a valid value meaning "no filtering"
	minKeyword = cfg.Search.DefaultMinKeywordScore
	minSemantic = cfg.Search.DefaultMinSemanticScore
	return minKeyword, minSemantic
}

// searchArgsReorder moves any flags (and their values) that appear after the query
// to the front of the slice so that flag.Parse() sees them. Go's flag package
// stops at the first non-flag argument, so "sagasu search \"query\" -min-score 0.5"
// would otherwise leave -min-score unparsed (default 0.49 used).
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
	searchArgs := searchArgsReorder(os.Args[2:])
	configPath := searchConfigPathFromArgs(searchArgs, defaultConfigPath)
	defaultMinKw, defaultMinSem := searchMinScoreDefaultsFromConfig(configPath)

	fs := flag.NewFlagSet("search", flag.ExitOnError)
	configPathFlag := fs.String("config", defaultConfigPath, "config file path")
	serverURL := fs.String("server", "http://localhost:8080", "server URL (empty = use direct storage when server is not running)")
	limit := fs.Int("limit", 10, "number of results")
	minKeywordScore := fs.Float64("min-keyword-score", defaultMinKw, "minimum score for keyword (non-semantic) results")
	minSemanticScore := fs.Float64("min-semantic-score", defaultMinSem, "minimum score for semantic-only results")
	kwEnabled := fs.Bool("keyword", true, "enable keyword search")
	semEnabled := fs.Bool("semantic", true, "enable semantic search")
	fuzzyEnabled := fs.Bool("fuzzy", false, "enable fuzzy matching for typo tolerance")
	outputFormat := fs.String("output", "text", "output format: text (human-readable), compact (one result per line), or json (parseable)")
	fs.Usage = func() { printSearchUsage(fs) }
	_ = fs.Parse(searchArgs)

	if fs.NArg() < 1 {
		printSearchUsage(fs)
		os.Exit(1)
	}
	queryStr := buildSearchQuery(fs.Args())
	if queryStr == "" {
		printSearchUsage(fs)
		os.Exit(1)
	}

	format := cli.OutputText
	switch *outputFormat {
	case "json":
		format = cli.OutputJSON
	case "text":
		format = cli.OutputText
	case "compact":
		format = cli.OutputCompact
	default:
		fmt.Printf("Unknown output format %q; use text, compact, or json\n", *outputFormat)
		os.Exit(1)
	}

	searchQuery := &models.SearchQuery{
		Query:            queryStr,
		Limit:            *limit,
		MinKeywordScore:  *minKeywordScore,
		MinSemanticScore: *minSemanticScore,
		KeywordEnabled:   *kwEnabled,
		SemanticEnabled:  *semEnabled,
		FuzzyEnabled:     *fuzzyEnabled,
	}

	if *serverURL != "" {
		// Use HTTP API when server is running (avoids Bleve/SQLite lock conflict).
		response, err := searchViaHTTP(*serverURL, searchQuery)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Search failed: %v\n", err)
			os.Exit(1)
		}
		// Auto-retry with fuzzy if no results and fuzzy not already enabled
		if !searchQuery.FuzzyEnabled && response.TotalNonSemantic == 0 && response.TotalSemantic == 0 {
			searchQuery.FuzzyEnabled = true
			fuzzyResponse, fuzzyErr := searchViaHTTP(*serverURL, searchQuery)
			if fuzzyErr == nil && (fuzzyResponse.TotalNonSemantic > 0 || fuzzyResponse.TotalSemantic > 0) {
				response = fuzzyResponse
				response.AutoFuzzy = true
			}
		}
		if err := cli.WriteSearchResults(os.Stdout, response, format); err != nil {
			fmt.Fprintf(os.Stderr, "Output failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Direct storage access (when server is not running).
	cfg, _, err := loadConfig(*configPathFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}
	debugMode := cfg.Debug
	logger, err := utils.NewLogger(debugMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	components, err := initializeComponents(cfg, logger, debugMode)
	if err != nil {
		logger.Fatal("Failed to initialize", zap.Error(err))
	}
	defer components.Close()

	response, err := components.Engine.Search(context.Background(), searchQuery)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Search failed: %v\n", err)
		os.Exit(1)
	}
	// Auto-retry with fuzzy if no results and fuzzy not already enabled
	if !searchQuery.FuzzyEnabled && response.TotalNonSemantic == 0 && response.TotalSemantic == 0 {
		searchQuery.FuzzyEnabled = true
		fuzzyResponse, fuzzyErr := components.Engine.Search(context.Background(), searchQuery)
		if fuzzyErr == nil && (fuzzyResponse.TotalNonSemantic > 0 || fuzzyResponse.TotalSemantic > 0) {
			response = fuzzyResponse
			response.AutoFuzzy = true
		}
	}
	if err := cli.WriteSearchResults(os.Stdout, response, format); err != nil {
		fmt.Fprintf(os.Stderr, "Output failed: %v\n", err)
		os.Exit(1)
	}
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

// statusConfigResponse holds configuration info returned by status.
type statusConfigResponse struct {
	VectorIndexType     string `json:"vector_index_type"`
	EmbeddingDimensions int    `json:"embedding_dimensions,omitempty"`
	ChunkSize           int    `json:"chunk_size,omitempty"`
	ChunkOverlap        int    `json:"chunk_overlap,omitempty"`
	RankingEnabled      bool   `json:"ranking_enabled,omitempty"`
	DatabasePath        string `json:"database_path,omitempty"`
	BleveIndexPath      string `json:"bleve_index_path,omitempty"`
	FAISSIndexPath      string `json:"faiss_index_path,omitempty"`
}

// statusResponse is the shape of GET /api/v1/status response.
type statusResponse struct {
	Documents       int64                 `json:"documents"`
	Chunks          int64                 `json:"chunks"`
	VectorIndexSize int                   `json:"vector_index_size"`
	DiskUsageBytes  *int64                `json:"disk_usage_bytes,omitempty"`
	Config          *statusConfigResponse `json:"config,omitempty"`
}

func runStatus() {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	configPath := fs.String("config", defaultConfigPath, "config file path")
	serverURL := fs.String("server", "http://localhost:8080", "server URL (empty = use direct storage)")
	outputFormat := fs.String("output", "text", "output format: text or json")
	_ = fs.Parse(os.Args[2:])

	var status statusResponse
	if *serverURL != "" {
		res, err := statusViaHTTP(*serverURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Status failed: %v\n", err)
			os.Exit(1)
		}
		status = *res
	} else {
		cfg, _, err := loadConfig(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			os.Exit(1)
		}
		debugMode := cfg.Debug
		logger, err := utils.NewLogger(debugMode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
			os.Exit(1)
		}
		defer logger.Sync()
		components, err := initializeComponents(cfg, logger, debugMode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize: %v\n", err)
			os.Exit(1)
		}
		defer components.Close()
		ctx := context.Background()
		docCount, err := components.Storage.CountDocuments(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Count documents failed: %v\n", err)
			os.Exit(1)
		}
		chunkCount, err := components.Storage.CountChunks(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Count chunks failed: %v\n", err)
			os.Exit(1)
		}
		status = statusResponse{
			Documents:       docCount,
			Chunks:          chunkCount,
			VectorIndexSize: components.Engine.VectorIndexSize(),
			Config: &statusConfigResponse{
				VectorIndexType:     components.Engine.VectorIndexType(),
				EmbeddingDimensions: cfg.Embedding.Dimensions,
				ChunkSize:           cfg.Search.ChunkSize,
				ChunkOverlap:        cfg.Search.ChunkOverlap,
				RankingEnabled:      cfg.Search.RankingEnabled,
				DatabasePath:        cfg.Storage.DatabasePath,
				BleveIndexPath:      cfg.Storage.BleveIndexPath,
				FAISSIndexPath:      cfg.Storage.FAISSIndexPath,
			},
		}
		diskBytes, err := storage.DiskUsageBytes(cfg.Storage.DatabasePath, cfg.Storage.BleveIndexPath, cfg.Storage.FAISSIndexPath)
		if err == nil {
			status.DiskUsageBytes = &diskBytes
		}
	}

	switch *outputFormat {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(status); err != nil {
			fmt.Fprintf(os.Stderr, "Output failed: %v\n", err)
			os.Exit(1)
		}
	case "text":
		fmt.Printf("documents:          %d   # count of indexed documents\n", status.Documents)
		fmt.Printf("chunks:             %d   # count of text chunks\n", status.Chunks)
		fmt.Printf("vector_index_size:  %d   # count of vectors in semantic index\n", status.VectorIndexSize)
		if status.DiskUsageBytes != nil {
			fmt.Printf("disk_usage_bytes:   %d   # storage + indices on disk\n", *status.DiskUsageBytes)
		}
		if status.Config != nil {
			fmt.Println()
			fmt.Println("# configuration")
			fmt.Printf("vector_index_type:  %s\n", status.Config.VectorIndexType)
			if status.Config.EmbeddingDimensions > 0 {
				fmt.Printf("embedding_dims:     %d\n", status.Config.EmbeddingDimensions)
			}
			if status.Config.ChunkSize > 0 {
				fmt.Printf("chunk_size:         %d\n", status.Config.ChunkSize)
			}
			if status.Config.ChunkOverlap > 0 {
				fmt.Printf("chunk_overlap:      %d\n", status.Config.ChunkOverlap)
			}
			fmt.Printf("ranking_enabled:    %t\n", status.Config.RankingEnabled)
			if status.Config.DatabasePath != "" {
				fmt.Printf("database_path:      %s\n", status.Config.DatabasePath)
			}
			if status.Config.BleveIndexPath != "" {
				fmt.Printf("bleve_index_path:   %s\n", status.Config.BleveIndexPath)
			}
			if status.Config.FAISSIndexPath != "" {
				fmt.Printf("faiss_index_path:   %s\n", status.Config.FAISSIndexPath)
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown output format %q; use text or json\n", *outputFormat)
		os.Exit(1)
	}
}

func statusViaHTTP(serverURL string) (*statusResponse, error) {
	resp, err := http.Get(serverURL + "/api/v1/status")
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(b))
	}
	var s statusResponse
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &s, nil
}

func runIndex() {
	fs := flag.NewFlagSet("index", flag.ExitOnError)
	configPath := fs.String("config", defaultConfigPath, "config file path")
	_ = fs.String("title", "", "document title (unused; document title is derived from filename)")
	_ = fs.Parse(os.Args[2:])

	if fs.NArg() < 1 {
		fmt.Println("Usage: sagasu index [flags] <file-or-directory>")
		os.Exit(1)
	}
	path := fs.Arg(0)

	cfg, _, err := loadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}
	debugMode := cfg.Debug
	logger, err := utils.NewLogger(debugMode)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	components, err := initializeComponents(cfg, logger, debugMode)
	if err != nil {
		logger.Fatal("Failed to initialize", zap.Error(err))
	}
	defer components.Close()

	ctx := context.Background()
	info, err := os.Stat(path)
	if err != nil {
		fmt.Printf("Failed to stat path: %v\n", err)
		os.Exit(1)
	}
	if info.IsDir() {
		exts := cfg.Watch.Extensions
		if exts == nil {
			exts = []string{".txt", ".md", ".rst", ".pdf", ".docx", ".xlsx"}
		}
		n, err := components.Indexer.IndexDirectory(ctx, path, exts)
		if err != nil {
			fmt.Printf("Indexing directory failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Indexed %d file(s) from %s\n", n, path)
		return
	}
	// Single file: no extension filter
	if err := components.Indexer.IndexFile(ctx, path, nil); err != nil {
		fmt.Printf("Indexing failed: %v\n", err)
		os.Exit(1)
	}
	absPath, _ := filepath.Abs(path)
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
	debugMode := cfg.Debug
	logger, err := utils.NewLogger(debugMode)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	components, err := initializeComponents(cfg, logger, debugMode)
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

func initializeComponents(cfg *config.Config, logger *zap.Logger, debug bool) (*Components, error) {
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

	vectorIndex, err := vector.NewVectorIndex(cfg.Vector.IndexType, cfg.Embedding.Dimensions)
	if err != nil {
		// Fall back to memory index if configured type fails (e.g., FAISS not available)
		if cfg.Vector.IndexType != "memory" && cfg.Vector.IndexType != "" {
			if logger != nil {
				logger.Warn("failed to create vector index, falling back to memory",
					zap.String("requested_type", cfg.Vector.IndexType),
					zap.Error(err))
			}
			vectorIndex, err = vector.NewVectorIndex("memory", cfg.Embedding.Dimensions)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize vector index: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to initialize vector index: %w", err)
		}
	}
	if cfg.Storage.FAISSIndexPath != "" {
		if loadErr := vectorIndex.Load(cfg.Storage.FAISSIndexPath); loadErr != nil && logger != nil {
			logger.Warn("vector index load skipped (use full sync)", zap.String("path", cfg.Storage.FAISSIndexPath), zap.Error(loadErr))
		}
	}
	if logger != nil {
		logger.Info("vector index initialized",
			zap.String("type", cfg.Vector.IndexType),
			zap.Bool("faiss_available", vector.IsFAISSAvailable()))
	}

	keywordIndex, err := keyword.NewBleveIndex(cfg.Storage.BleveIndexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize keyword index: %w", err)
	}

	engine := search.NewEngine(store, embedder, vectorIndex, keywordIndex, &cfg.Search)
	// Initialize spell checker for typo tolerance
	engine.WithSpellChecker()

	idxOpts := []indexer.IndexerOption{}
	if debug && logger != nil {
		idxOpts = append(idxOpts, indexer.WithLogger(logger))
	}
	idx := indexer.NewIndexer(store, embedder, vectorIndex, keywordIndex, &cfg.Search, extract.NewExtractor(), idxOpts...)

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
  sagasu status [flags]           Show engine/storage/index status
  sagasu watch <add|remove|list>  Manage watched directories
  sagasu version                  Show version
  sagasu help                     Show this help

Server Flags:
  --config string    Config file path (default: /usr/local/etc/sagasu/config.yaml)
  --debug            Enable debug logging (directory changes, file indexing, etc.)

Search Flags:
  --config string             Config file path (for direct storage mode; also used for default min-score values)
  --server string             Server URL (default: http://localhost:8080). Use empty (--server "") to use direct storage when server is not running.
  --limit int                 Number of results per list (default: 10)
  --min-keyword-score float   Minimum score for keyword results (default from config, or 0.49)
  --min-semantic-score float  Minimum score for semantic-only results (default from config, or 0.49)
  --keyword                   Enable keyword search (default: true)
  --semantic                  Enable semantic search (default: true)
  --fuzzy                     Enable fuzzy matching for typo tolerance (default: false)

Index Flags:
  --config string    Config file path
  --title string     Document title

Status Flags:
  --config string    Config file path (for direct storage mode)
  --server string    Server URL (default: http://localhost:8080). Use empty (--server "") for direct storage.
  --output string    Output format: text or json (default: text)

Watch Flags:
  --server string    Server URL (default: http://localhost:8080)

Examples:
  sagasu server
  sagasu search "machine learning algorithms"
  sagasu search --min-keyword-score 0.1 "raosan"
  sagasu search --output json "query"   # structured JSON for other apps
  sagasu search --keyword=false "neural networks"   # semantic-only
  sagasu index --title "My Document" document.txt
  sagasu delete doc-123
  sagasu status
  sagasu status --output json
  sagasu watch add /path/to/docs
  sagasu watch list`)
}
