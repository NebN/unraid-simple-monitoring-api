package util

import (
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const DECIMAL_KILO = 1000
const BINARY_KILO = 1024

const (
	BIT  = "b"
	BYTE = "B"

	KILO   = "K"
	MEGA   = "M"
	GIGA   = "G"
	TERA   = "T"
	PETA   = "P"
	EXA    = "E"
	ZETTA  = "Z"
	YOTTA  = "Y"
	RONNA  = "R"
	QUETTA = "Q"

	KIBI  = "Ki"
	MEBI  = "Mi"
	GIBI  = "Gi"
	TEBI  = "Ti"
	EXBI  = "Ei"
	PEBI  = "Pi"
	ZEBI  = "Zi"
	YOBI  = "Yi"
	ROBI  = "Ri"
	QUEBI = "Qi"
)

type UnitPrefix struct {
	label    string
	exponent int
	bytes    uint64
	isBinary bool
}

func NewUnitPrefix(label string, exponent int) UnitPrefix {
	var base float64
	isBinary := strings.HasSuffix(label, "i")
	if isBinary {
		base = BINARY_KILO
	} else {
		base = DECIMAL_KILO
	}
	return UnitPrefix{
		label:    label,
		exponent: exponent,
		bytes:    uint64(math.Pow(base, float64(exponent))),
		isBinary: isBinary,
	}
}

var (
	unitMap = map[string]UnitPrefix{
		BYTE:   NewUnitPrefix(BYTE, 0),
		KILO:   NewUnitPrefix(KILO, 1),
		MEGA:   NewUnitPrefix(MEGA, 2),
		GIGA:   NewUnitPrefix(GIGA, 3),
		TERA:   NewUnitPrefix(TERA, 4),
		PETA:   NewUnitPrefix(PETA, 5),
		EXA:    NewUnitPrefix(EXA, 6),
		ZETTA:  NewUnitPrefix(ZETTA, 7),
		YOTTA:  NewUnitPrefix(YOTTA, 8),
		RONNA:  NewUnitPrefix(RONNA, 9),
		QUETTA: NewUnitPrefix(QUETTA, 10),
		KIBI:   NewUnitPrefix(KIBI, 1),
		MEBI:   NewUnitPrefix(MEBI, 2),
		GIBI:   NewUnitPrefix(GIBI, 3),
		TEBI:   NewUnitPrefix(TEBI, 4),
		PEBI:   NewUnitPrefix(PEBI, 5),
		EXBI:   NewUnitPrefix(EXA, 6),
		ZEBI:   NewUnitPrefix(ZEBI, 7),
		YOBI:   NewUnitPrefix(YOBI, 8),
		ROBI:   NewUnitPrefix(ROBI, 9),
		QUEBI:  NewUnitPrefix(QUEBI, 10),
	}
)

var (
	ratioMap map[string]float64
)

func init() {
	ratioMap = make(map[string]float64)
	base := func(unit UnitPrefix) float64 {
		if unit.isBinary {
			return BINARY_KILO
		} else {
			return DECIMAL_KILO
		}
	}

	for _, fromUnit := range unitMap {
		for _, toUnit := range unitMap {
			// if they are both binary or decimal, we do not require the ratios
			if fromUnit.isBinary != toUnit.isBinary {
				key := fromUnit.label + toUnit.label
				ratio := math.Pow(base(fromUnit), float64(fromUnit.exponent)) / math.Pow(base(toUnit), float64(toUnit.exponent))
				ratioMap[key] = ratio
			}
		}
	}
}

func SizeConvertionFunction(fromUnitLabel string, toUnitLabel string) func(float64) float64 {
	fromUnit := unitMap[fromUnitLabel]
	toUnit := unitMap[toUnitLabel]

	if fromUnit.isBinary == toUnit.isBinary {
		return func(value float64) float64 {
			if fromUnit.isBinary {
				shift := (fromUnit.exponent - toUnit.exponent) * 10
				mantissa, exponent := math.Frexp(value)
				return math.Ldexp(mantissa, exponent+shift)
			} else {
				shift := (fromUnit.exponent - toUnit.exponent) * 3
				return value * math.Pow10(shift)
			}

		}
	} else {
		return func(value float64) float64 {
			round := func(v float64) float64 {
				return v
			}
			if toUnit.label == "B" {
				round = func(v float64) float64 {
					// fixing floating point math when going to Bytes
					return math.Round(v)
				}
			}
			return round(value * ratioMap[fromUnit.label+toUnit.label])
		}
	}
}

func BytesToMegaBytes(b float64) float64 {
	return SizeConvertionFunction(BYTE, MEGA)(b)
}

func BytesToMebiBytes(b float64) float64 {
	return SizeConvertionFunction(BYTE, MEBI)(b)
}

func BytesToGibiBytes(b float64) float64 {
	return SizeConvertionFunction(BYTE, GIBI)(b)
}

func KibiBytesToMebiBytes(b float64) float64 {
	return SizeConvertionFunction(KIBI, MEBI)(b)
}

func BytesToBits(b float64) float64 {
	mantissa, exponent := math.Frexp(b)
	return math.Ldexp(mantissa, exponent+3)
}

var (
	zfsSizeRegex = regexp.MustCompile(`(\d+.?\d+?)([BKMGTPZYRQ])`)
)

func ParseZfsSize(size string) (float64, error) {
	res := zfsSizeRegex.FindStringSubmatch(size)
	if len(res) > 1 {
		number := res[1]
		unit := res[2]
		if unit != "B" {
			unit = unit + "i" // it's a binary unit, so G (GigaBytes) becomes Gi (GibiBytes)
		}
		parsed, err := strconv.ParseFloat(number, 64)

		if err != nil {
			slog.Error("Unable to parse zfs size", slog.String("raw value", number))
			return 0, err
		}

		return parsed * float64(unitMap[unit].bytes), nil
	} else {
		slog.Error("Unable to parse zfs size", slog.String("raw value", size))
		return 0, fmt.Errorf("unable to match size from string %s", size)
	}
}
