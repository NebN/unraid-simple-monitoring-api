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

func floatAreEqualEnough(a float64, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
