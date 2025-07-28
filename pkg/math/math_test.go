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

func TestModulo(t *testing.T) {
	type test struct {
		a, b        int
		expected    int
		errExpected bool
	}
	tests := []test{
		{10, 3, 1, false},
		{-10, 3, -1, false},
		{10, -3, 1, false},
		{-10, -3, -1, false},
		{10, 0, 0, true},
		{0, 10, 0, false},
		{100, 10, 0, false},
		{1000, 100, 0, false},
		{1001, 100, 1, false},
	}
	for i, tt := range tests {
		got, err := Modulo(tt.a, tt.b)
		if (err != nil) != tt.errExpected {
			t.Errorf("Test %d: Modulo(%v, %v) error = %v; wantErr %v", i, tt.a, tt.b, err, tt.errExpected)
			continue
		}
		if !tt.errExpected && got != tt.expected {
			t.Errorf("Test %d: Modulo(%v, %v) = %v; want %v", i, tt.a, tt.b, got, tt.expected)
		}
	}
}

func Test_ExampleCases(t *testing.T) {
	fmt.Println("Example test case for Add:", Add(10, 20))
	fmt.Println("Example test case for Subtract:", Subtract(20, 10))
	fmt.Println("Example test case for Multiply:", Multiply(5, 5))
	res, err := Divide(10, 2)
	if err != nil {
		fmt.Println("Error in Divide:", err)
	} else {
		fmt.Println("Example test case for Divide:", res)
	}
	fmt.Println("Example test case for Power:", Power(2, 3))
	res1, err1 := Modulo(10, 3)
	if err1 != nil {
		fmt.Println("Error in Modulo:", err1)
	} else {
		fmt.Println("Example test case for Modulo:", res1)
	}
}
