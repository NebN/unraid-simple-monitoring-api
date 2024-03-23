package util

import "math"

func BytesToMegaBytes(b float64) float64 {
	return b / 1_000_000
}

func BytesToMebiBytes(b float64) float64 {
	mantissa, exponent := math.Frexp(b)
	return math.Ldexp(mantissa, exponent-20)
}

func BytesToBits(b float64) float64 {
	mantissa, exponent := math.Frexp(b)
	return math.Ldexp(mantissa, exponent+3)
}

func BytesToGibiBytes(b uint64) uint64 {
	return b >> 30
}
