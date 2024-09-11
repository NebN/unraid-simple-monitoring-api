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
		t.Fatalf("expected: %f, got: %f", expected, res)
	}
}

func TestBytesToMebiBytes(t *testing.T) {
	bytes := 12_000_000.0
	expected := 11.444091797

	res := BytesToMebiBytes(bytes)
	if !floatAreEqualEnough(res, expected) {
		t.Fatalf("expected: %f, got: %f", expected, res)
	}
}

func TestBytesToBits(t *testing.T) {
	bytes := 12_000_000.0
	expected := 96000000.0

	res := BytesToBits(bytes)
	if !floatAreEqualEnough(res, expected) {
		t.Fatalf("expected: %f, got: %f", expected, res)
	}
}

func TestBytesToGibiBytes(t *testing.T) {
	// BytesToGibiBytes simply bitshifts, so it truncates instead of rounding
	// this is so far the expected behaviour
	// 6,442,000,000 is slightly less than 6GiB, we get 5
	// 6,444,000,000 is slightly more than 6GiB, we get 6

	var bytes uint64 = 644_2_000000
	var expected uint64 = 5

	var res = BytesToGibiBytes(bytes)
	if res != expected {
		t.Fatalf("expected: %d, got: %d", expected, res)
	}

	bytes = 644_4_000000
	expected = 6

	res = BytesToGibiBytes(bytes)
	if res != expected {
		t.Fatalf("expected: %d, got: %d", expected, res)
	}
}

func TestRoundTwoDecimals(t *testing.T) {
	number := 123.456
	expected := 123.46
	res := RoundTwoDecimals(number)
	if !floatAreEqualEnough(res, expected) {
		t.Fatalf("expected: %f, got: %f", expected, res)
	}

	number = 123.454
	expected = 123.45
	res = RoundTwoDecimals(number)
	if !floatAreEqualEnough(res, expected) {
		t.Fatalf("expected: %f, got: %f", expected, res)
	}

	number = 123.3
	expected = 123.30
	res = RoundTwoDecimals(number)
	if !floatAreEqualEnough(res, expected) {
		t.Fatalf("expected: %f, got: %f", expected, res)
	}
}

func TestParseZfsSize(t *testing.T) {
	str := "12.5G"
	var expected uint64 = 13421772800
	res, err := ParseZfsSize(str)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != expected {
		t.Fatalf("expected: %d, got: %d", expected, res)
	}

	str = "230.5T"
	expected = 253437430202368
	res, err = ParseZfsSize(str)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != expected {
		t.Fatalf("expected: %d, got: %d", expected, res)
	}

	str = "130.50M"
	expected = 136839168
	res, err = ParseZfsSize(str)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != expected {
		t.Fatalf("expected: %d, got: %d", expected, res)
	}

	str = "93K"
	expected = 95232
	res, err = ParseZfsSize(str)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != expected {
		t.Fatalf("expected: %d, got: %d", expected, res)
	}

	str = "100B"
	expected = 100
	res, err = ParseZfsSize(str)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != expected {
		t.Fatalf("expected: %d, got: %d", expected, res)
	}
}

func floatAreEqualEnough(a float64, b float64) bool {
	return math.Abs(a-b) < 1e-9
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
