package utils

import "testing"

func TestCosine(t *testing.T) {
	if got := Cosine([]float64{1, 0}, []float64{1, 0}); got < 0.999 {
		t.Fatalf("same vector cosine too low: %v", got)
	}
	if got := Cosine([]float64{1, 0}, []float64{0, 1}); got != 0 {
		t.Fatalf("orthogonal cosine wrong: %v", got)
	}
}
