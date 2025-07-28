package math

import (
	"errors"
	
)

// Add returns the sum of two numbers
func Add(a, b float64) float64 {
	return a + b
}

// Subtract returns the difference of two numbers
func Subtract(a, b float64) float64 {
	return a - b
}

// Multiply returns the product of two numbers
func Multiply(a, b float64) float64 {
	return a * b
}

// Divide returns the quotient of two numbers
func Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, errors.New("division by zero is not allowed")
	}
	return a / b, nil
}

// Power returns a raised to the power of b
func Power(a, b float64) float64 {
	result := 1.0
	for i := 0; i < int(b); i++ {
		result *= a
	}
	return result
}

// Modulo returns the remainder of a divided by b
func Modulo(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("modulo by zero is not allowed")
	}
	return a % b, nil
}

