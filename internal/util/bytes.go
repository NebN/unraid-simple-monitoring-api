package util

import (
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strconv"
)

var zfsSizeRegex = regexp.MustCompile(`(\d+.?\d+?)([BKMGTPY])`)

const BINARY_KILO = 1024

var unitMultiplier = map[string]uint64{
	"B": uint64(math.Pow(BINARY_KILO, 0)),
	"K": uint64(math.Pow(BINARY_KILO, 1)),
	"M": uint64(math.Pow(BINARY_KILO, 2)),
	"G": uint64(math.Pow(BINARY_KILO, 3)),
	"T": uint64(math.Pow(BINARY_KILO, 4)),
	"P": uint64(math.Pow(BINARY_KILO, 5)),
	"Y": uint64(math.Pow(BINARY_KILO, 6)),
}

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

func KibiBytesToMebiBytes(b uint64) uint64 {
	return b >> 10
}

func ParseZfsSize(size string) (uint64, error) {
	res := zfsSizeRegex.FindStringSubmatch(size)
	if len(res) > 1 {
		number := res[1]
		unit := res[2]
		parsed, err := strconv.ParseFloat(number, 64)

		if err != nil {
			slog.Error("Unable to parse zfs size", slog.String("raw value", number))
			return 0, err
		}

		return uint64(parsed * float64(unitMultiplier[unit])), nil
	} else {
		slog.Error("Unable to parse zfs size", slog.String("raw value", size))
		return 0, fmt.Errorf("unable to match size from string %s", size)
	}
}
