package testdemo

import "errors"

// Add performs addition of two integers
func Add(a, b int) int {
    return a + b
}

// Subtract performs subtraction of two integers  
func Subtract(a, b int) int {
    return a - b
}

// Divide performs division with error handling
func Divide(a, b int) (float64, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return float64(a) / float64(b), nil
}

// IsPositive checks if a number is positive
func IsPositive(n int) bool {
    return n > 0
}
