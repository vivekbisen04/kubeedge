package calculator

import (
    "errors"
    "math"
)

// Multiply performs multiplication of two numbers
func Multiply(a, b float64) float64 {
    return a * b
}

// Power calculates a to the power of b
func Power(base, exponent float64) float64 {
    return math.Pow(base, exponent)
}

// SquareRoot calculates the square root with error handling
func SquareRoot(n float64) (float64, error) {
    if n < 0 {
        return 0, errors.New("cannot calculate square root of negative number")
    }
    return math.Sqrt(n), nil
}

// Factorial calculates factorial of a positive integer
func Factorial(n int) (int, error) {
    if n < 0 {
        return 0, errors.New("factorial of negative number is undefined")
    }
    if n == 0 || n == 1 {
        return 1, nil
    }
    
    result := 1
    for i := 2; i <= n; i++ {
        result *= i
    }
    return result, nil
}

// Average calculates the average of a slice of numbers
func Average(numbers []float64) (float64, error) {
    if len(numbers) == 0 {
        return 0, errors.New("cannot calculate average of empty slice")
    }
    
    sum := 0.0
    for _, num := range numbers {
        sum += num
    }
    return sum / float64(len(numbers)), nil
}
