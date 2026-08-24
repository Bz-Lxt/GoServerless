package metrics

import "testing"

func TestPercentile(t *testing.T) {
	if Percentile(nil, 0.95) != 0 {
		t.Fatal("empty")
	}
	vals := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	p50 := Percentile(vals, 0.5)
	if p50 < 5 || p50 > 6 {
		t.Fatalf("p50=%v", p50)
	}
	p100 := Percentile(vals, 1)
	if p100 != 10 {
		t.Fatalf("p100=%v", p100)
	}
}

func TestMean(t *testing.T) {
	if Mean([]float64{10, 20, 30}) != 20 {
		t.Fatal("mean")
	}
}
