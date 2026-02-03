//go:build cgo
// +build cgo

// Package embedding provides ONNX-based embedding (requires CGO and onnxruntime library).
package embedding

import (
	"context"
	"fmt"
	"math"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

// ONNXEmbedder uses ONNX Runtime to produce embeddings. It requires CGO and the onnxruntime shared library.
type ONNXEmbedder struct {
	session    *ort.AdvancedSession
	dimensions int
	maxTokens  int
	cache      *EmbeddingCache
	tokenizer  Tokenizer
	// Pre-allocated tensors for Run(); we update input data and read output.
	inputIDsTensor      *ort.Tensor[int64]
	attentionMaskTensor *ort.Tensor[int64]
	tokenTypeIDsTensor  *ort.Tensor[int64]
	outputTensor        *ort.Tensor[float32]
	mu                  sync.Mutex
}

// NewONNXEmbedder creates an ONNX embedder. InitializeEnvironment is called if not already done.
func NewONNXEmbedder(modelPath string, dimensions, maxTokens, cacheSize int) (*ONNXEmbedder, error) {
	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX runtime: %w", err)
	}

	tokenizer := &SimpleTokenizer{}
	inputIDs, attentionMask, tokenTypeIDs := tokenizer.Tokenize("", maxTokens)

	inputIDsTensor, err := ort.NewTensor(ort.NewShape(1, int64(maxTokens)), inputIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	attentionMaskTensor, err := ort.NewTensor(ort.NewShape(1, int64(maxTokens)), attentionMask)
	if err != nil {
		inputIDsTensor.Destroy()
		return nil, fmt.Errorf("failed to create attention_mask tensor: %w", err)
	}
	tokenTypeIDsTensor, err := ort.NewTensor(ort.NewShape(1, int64(maxTokens)), tokenTypeIDs)
	if err != nil {
		inputIDsTensor.Destroy()
		attentionMaskTensor.Destroy()
		return nil, fmt.Errorf("failed to create token_type_ids tensor: %w", err)
	}
	outputData := make([]float32, dimensions)
	outputTensor, err := ort.NewTensor(ort.NewShape(1, int64(dimensions)), outputData)
	if err != nil {
		inputIDsTensor.Destroy()
		attentionMaskTensor.Destroy()
		tokenTypeIDsTensor.Destroy()
		return nil, fmt.Errorf("failed to create output tensor: %w", err)
	}

	inputs := []ort.ArbitraryTensor{inputIDsTensor, attentionMaskTensor, tokenTypeIDsTensor}
	outputs := []ort.ArbitraryTensor{outputTensor}
	session, err := ort.NewAdvancedSession(
		modelPath,
		[]string{"input_ids", "attention_mask", "token_type_ids"},
		[]string{"output"},
		inputs,
		outputs,
		nil,
	)
	if err != nil {
		inputIDsTensor.Destroy()
		attentionMaskTensor.Destroy()
		tokenTypeIDsTensor.Destroy()
		outputTensor.Destroy()
		return nil, fmt.Errorf("failed to create ONNX session: %w", err)
	}

	return &ONNXEmbedder{
		session:             session,
		dimensions:          dimensions,
		maxTokens:           maxTokens,
		cache:               NewEmbeddingCache(cacheSize),
		tokenizer:           tokenizer,
		inputIDsTensor:      inputIDsTensor,
		attentionMaskTensor: attentionMaskTensor,
		tokenTypeIDsTensor:  tokenTypeIDsTensor,
		outputTensor:        outputTensor,
	}, nil
}

// Embed returns the embedding for text, using cache when available.
func (e *ONNXEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if cached, ok := e.cache.Get(text); ok {
		return cached, nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	inputIDs, attentionMask, tokenTypeIDs := e.tokenizer.Tokenize(text, e.maxTokens)

	copy(e.inputIDsTensor.GetData(), inputIDs)
	copy(e.attentionMaskTensor.GetData(), attentionMask)
	copy(e.tokenTypeIDsTensor.GetData(), tokenTypeIDs)

	if err := e.session.Run(); err != nil {
		return nil, fmt.Errorf("inference failed: %w", err)
	}

	outputData := e.outputTensor.GetData()
	embedding := make([]float32, e.dimensions)
	copy(embedding, outputData[:e.dimensions])

	NormalizeL2Slice(embedding)
	e.cache.Set(text, embedding)
	return embedding, nil
}

// EmbedBatch calls Embed for each text.
func (e *ONNXEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := e.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		embeddings[i] = emb
	}
	return embeddings, nil
}

// Dimensions returns the embedding dimension.
func (e *ONNXEmbedder) Dimensions() int {
	return e.dimensions
}

// Close destroys the session and tensors.
func (e *ONNXEmbedder) Close() error {
	var err error
	if e.session != nil {
		err = e.session.Destroy()
		e.session = nil
	}
	if e.inputIDsTensor != nil {
		_ = e.inputIDsTensor.Destroy()
		e.inputIDsTensor = nil
	}
	if e.attentionMaskTensor != nil {
		_ = e.attentionMaskTensor.Destroy()
		e.attentionMaskTensor = nil
	}
	if e.tokenTypeIDsTensor != nil {
		_ = e.tokenTypeIDsTensor.Destroy()
		e.tokenTypeIDsTensor = nil
	}
	if e.outputTensor != nil {
		_ = e.outputTensor.Destroy()
		e.outputTensor = nil
	}
	return err
}

// NormalizeL2Slice normalizes the slice in place to unit L2 norm.
func NormalizeL2Slice(x []float32) {
	var sum float32
	for _, v := range x {
		sum += v * v
	}
	if sum == 0 {
		return
	}
	norm := float32(1.0 / math.Sqrt(float64(sum)))
	for i := range x {
		x[i] *= norm
	}
}
