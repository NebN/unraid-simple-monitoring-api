package util

import (
	"math"
)

type Float interface {
	~float64
}

func RoundNDecimals[T Float](n int) func(T) T {
	scale := math.Pow10(n)
	return func(v T) T {
		return T(math.Round(float64(v)*scale) / scale)
	}
}

func RoundTwoDecimals[T Float](n T) T {
	return RoundNDecimals[T](2)(n)
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
