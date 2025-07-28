package math

import (
	"fmt"
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
	for i, tt := range tests {
		got := Add(tt.a, tt.b)
		if got != tt.expected {
			t.Errorf("Test %d: Add(%v, %v) = %v; want %v", i, tt.a, tt.b, got, tt.expected)
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
	for i, tt := range tests {
		got := Subtract(tt.a, tt.b)
		if got != tt.expected {
			t.Errorf("Test %d: Subtract(%v, %v) = %v; want %v", i, tt.a, tt.b, got, tt.expected)
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
	for i, tt := range tests {
		got := Multiply(tt.a, tt.b)
		if got != tt.expected {
			t.Errorf("Test %d: Multiply(%v, %v) = %v; want %v", i, tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestDivide(t *testing.T) {
	type test struct {
		a, b        float64
		expected    float64
		errExpected bool
	}
	tests := []test{
		{1, 2, 0.5, false},
		{-1, 2, -0.5, false},
		{1, -2, -0.5, false},
		{-1, -2, 0.5, false},
		{0, 1, 0, false},
		{1, 0, 0, true},
		{math.MaxFloat64, 2, math.MaxFloat64 / 2, false},                         //test with max float64
		{math.SmallestNonzeroFloat64, 2, math.SmallestNonzeroFloat64 / 2, false}, //test with smallest non-zero float64

	}
	for i, tt := range tests {
		got, err := Divide(tt.a, tt.b)
		if (err != nil) != tt.errExpected {
			t.Errorf("Test %d: Divide(%v, %v) error = %v; wantErr %v", i, tt.a, tt.b, err, tt.errExpected)
			continue
		}
		if !tt.errExpected && got != tt.expected {
			t.Errorf("Test %d: Divide(%v, %v) = %v; want %v", i, tt.a, tt.b, got, tt.expected)
		}
	}
}

func Test_Example(t *testing.T) {
	fmt.Println("Example test")
}
