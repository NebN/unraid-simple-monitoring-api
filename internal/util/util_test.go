package util

import (
	"math"
	"testing"
)

func TestBytesToMegaBytes(t *testing.T) {
	bytes := 12_000_000.0
	expected := 12.0

	res := BytesToMegaBytes(bytes)
	if !floatAreEqualEnough(res, expected) {
		t.Fatalf("expected: %.17f, got: %.17f", expected, res)
	}
}

func TestBytesToMebiBytes(t *testing.T) {
	bytes := 12_000_000.0
	expected := 11.444091797

	res := BytesToMebiBytes(bytes)
	if !floatAreEqualEnough(res, expected) {
		t.Fatalf("expected: %.17f, got: %.17f", expected, res)
	}
}

func TestBytesToBits(t *testing.T) {
	bytes := 12_000_000.0
	expected := 96000000.0

	res := BytesToBits(bytes)
	if !floatAreEqualEnough(res, expected) {
		t.Fatalf("expected: %.17f, got: %.17f", expected, res)
	}
}

func TestBytesToGibiBytes(t *testing.T) {
	var bytes float64 = 644_2_000000
	var expected float64 = 5.999580025673

	var res = BytesToGibiBytes(bytes)
	if !floatAreEqualEnough(res, expected) {
		t.Fatalf("expected: %.17f, got: %.17f", expected, res)
	}

	bytes = 644_4_000000
	expected = 6.001442670822

	res = BytesToGibiBytes(bytes)
	if !floatAreEqualEnough(res, expected) {
		t.Fatalf("expected: %.17f, got: %.17f", expected, res)
	}
}

func TestRoundTwoDecimals(t *testing.T) {
	number := 123.456
	expected := 123.46
	res := RoundTwoDecimals(number)
	if res != expected {
		t.Fatalf("expected: %.17f, got: %.17f", expected, res)
	}

	number = 123.454
	expected = 123.45
	res = RoundTwoDecimals(number)
	if res != expected {
		t.Fatalf("expected: %.17f, got: %.17f", expected, res)
	}

	number = 123.3
	expected = 123.30
	res = RoundTwoDecimals(number)
	if res != expected {
		t.Fatalf("expected: %.17f, got: %.17f", expected, res)
	}
}

func TestParseZfsSize(t *testing.T) {
	str := "12.5G"
	var expected float64 = 13421772800.0
	res, err := ParseZfsSize(str)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != expected {
		t.Fatalf("expected: %.17f, got: %.17f", expected, res)
	}

	str = "230.5T"
	expected = 253437430202368
	res, err = ParseZfsSize(str)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != expected {
		t.Fatalf("expected: %.17f, got: %.17f", expected, res)
	}

	str = "130.50M"
	expected = 136839168
	res, err = ParseZfsSize(str)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != expected {
		t.Fatalf("expected: %.17f, got: %.17f", expected, res)
	}

	str = "93K"
	expected = 95232
	res, err = ParseZfsSize(str)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != expected {
		t.Fatalf("expected: %.17f, got: %.17f", expected, res)
	}

	str = "100B"
	expected = 100
	res, err = ParseZfsSize(str)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != expected {
		t.Fatalf("expected: %.17f, got: %.17f", expected, res)
	}
}

func TestSizeConvertionFunction(t *testing.T) {
	tests := []struct {
		from     string
		to       string
		input    float64
		expected float64
	}{
		{"M", "G", 1000.0, 1.0},
		{"G", "M", 1.0, 1000.0},
		{"K", "M", 1000.0, 1.0},

		{"Mi", "Gi", 1024.0, 1.0},
		{"Gi", "Mi", 1.0, 1024.0},
		{"Ki", "Mi", 1024.0, 1.0},
		{"Gi", "Ki", 2.8, 2936012.8},

		{"G", "Gi", 1.0, 0.93132257461548},
		{"M", "Gi", 1600.0, 1.490116119},
		{"G", "Ki", 1.11, 1083984.375},

		{"Gi", "G", 32.0, 34.359738368},
		{"Mi", "M", 1.78, 1.86646528},
		{"Y", "Pi", 0.522200000262, 463806771.00010812},

		{"M", "M", 123.45, 123.45},
		{"Mi", "Mi", 123.45, 123.45},

		{"Gi", "B", 1.760422268882, 1890239018},
		{"G", "B", 77.973234914, 77973234914},
	}

	for _, tc := range tests {
		f := SizeConvertionFunction(tc.from, tc.to)
		result := f(tc.input)
		if !floatAreEqualEnough(result, tc.expected) {
			t.Errorf("conversion from %s to %s failed: input=%.17f expected=%.17f got=%.17f",
				tc.from, tc.to, tc.input, tc.expected, result)
		}
	}
}

func TestCommonBase(t *testing.T) {
	a := "thisPartIsInCommonX"
	b := "thisPartIsInCommonY"
	expected := "thisPartIsInCommon"

	res := CommonBase(a, b)

	if res != expected {
		t.Fatalf("expected: %s, got: %s", expected, res)
	}
}

func TestCommonBaseEmpty(t *testing.T) {
	a := "not empty"
	b := ""
	c := "not quite empty"
	expected := ""

	res := CommonBase(a, b, c)

	if res != expected {
		t.Fatalf("expected: %s, got: %s", expected, res)
	}
}

func TestCommonBaseIdentical(t *testing.T) {
	a := "identical"
	b := "identical"
	expected := "identical"

	res := CommonBase(a, b)

	if res != expected {
		t.Fatalf("expected: %s, got: %s", expected, res)
	}
}

func TestCommonBaseMultiple(t *testing.T) {
	a := "/common/part/ends/here/abc"
	b := "/common/part/ends/here/bcd"
	c := "/common/part/ends/here/cde"

	expected := "/common/part/ends/here/"

	res := CommonBase(a, b, c)

	if res != expected {
		t.Fatalf("expected: %s, got: %s", expected, res)
	}
}

func TestCommonBaseSingle(t *testing.T) {
	a := "single"

	expected := "single"

	res := CommonBase(a)

	if res != expected {
		t.Fatalf("expected: %s, got: %s", expected, res)
	}
}

func TestCommonBaseNothing(t *testing.T) {

	expected := ""

	res := CommonBase()

	if res != expected {
		t.Fatalf("expected: %s, got: %s", expected, res)
	}
}

func floatAreEqualEnough(a float64, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
