package utils

import "math"

// NormalizeL2 normalizes the slice in place to unit L2 norm.
// If the norm is zero, the slice is unchanged.
func NormalizeL2(x []float32) {
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
