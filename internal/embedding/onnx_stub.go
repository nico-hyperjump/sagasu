//go:build !cgo
// +build !cgo

package embedding

import (
	"errors"
)

// ONNXEmbedder stub type when built without CGO (see onnx.go for real implementation).
type ONNXEmbedder struct{}

// NewONNXEmbedder returns an error when built without CGO (ONNX not available).
func NewONNXEmbedder(_ string, _, _, _ int) (*ONNXEmbedder, error) {
	return nil, errors.New("ONNX embedder requires CGO; build with CGO_ENABLED=1 and onnxruntime")
}
