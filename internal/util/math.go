package util

import "math"

func RoundTwoDecimals(n float64) float64 {
	return math.Round(n*100) / 100
}

func Average(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}

	total := 0.0
	for _, v := range xs {
		total += v
	}
	return total / float64(len(xs))
}
