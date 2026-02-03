// Package vector provides similarity helpers for normalized vectors.
package vector

import "math"

// InnerProduct returns the inner product of two vectors (for normalized vectors equals cosine similarity).
func InnerProduct(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot float64
	for i := range a {
		dot += float64(a[i] * b[i])
	}
	return dot
}

// L2Norm returns the L2 norm of a vector.
func L2Norm(x []float32) float64 {
	var sum float64
	for _, v := range x {
		sum += float64(v * v)
	}
	return math.Sqrt(sum)
}
