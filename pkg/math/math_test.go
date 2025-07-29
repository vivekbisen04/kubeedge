package math

import (
	"math"
	"testing"
)

func TestAdd(t *testing.T) {
	type test struct {
		a, b, expected float64
	}
	tests := []test{
		{1, 2, 3},
		{-1, 2, 1},
		{1, -2, -1},
		{-1, -2, -3},
		{0, 0, 0},
		{math.MaxFloat64, math.MaxFloat64, math.Inf(1)},                                             //test with max float64
		{math.SmallestNonzeroFloat64, math.SmallestNonzeroFloat64, 2 * math.SmallestNonzeroFloat64}, //test with smallest non-zero float64

	}
	for _, tc := range tests {
		got := Add(tc.a, tc.b)
		if got != tc.expected {
			t.Errorf("Add(%v, %v) = %v; want %v", tc.a, tc.b, got, tc.expected)
		}
	}
}

func TestSubtract(t *testing.T) {
	type test struct {
		a, b, expected float64
	}
	tests := []test{
		{1, 2, -1},
		{-1, 2, -3},
		{1, -2, 3},
		{-1, -2, 1},
		{0, 0, 0},
		{math.MaxFloat64, 1, math.MaxFloat64 - 1},                     //test with max float64
		{math.SmallestNonzeroFloat64, math.SmallestNonzeroFloat64, 0}, //test with smallest non-zero float64

	}
	for _, tc := range tests {
		got := Subtract(tc.a, tc.b)
		if got != tc.expected {
			t.Errorf("Subtract(%v, %v) = %v; want %v", tc.a, tc.b, got, tc.expected)
		}
	}
}

func TestMultiply(t *testing.T) {
	type test struct {
		a, b, expected float64
	}
	tests := []test{
		{1, 2, 2},
		{-1, 2, -2},
		{1, -2, -2},
		{-1, -2, 2},
		{0, 0, 0},
		{math.MaxFloat64, 2, math.Inf(1)},   //test with max float64
		{math.SmallestNonzeroFloat64, 0, 0}, //test with smallest non-zero float64

	}
	for _, tc := range tests {
		got := Multiply(tc.a, tc.b)
		if got != tc.expected {
			t.Errorf("Multiply(%v, %v) = %v; want %v", tc.a, tc.b, got, tc.expected)
		}
	}
}
