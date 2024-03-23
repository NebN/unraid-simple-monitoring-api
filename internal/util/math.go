package util

import "math"

func RoundTwoDecimals(n float64) float64 {
	return math.Round(n*100) / 100
}
