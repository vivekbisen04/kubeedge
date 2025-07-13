package main

import (
    "fmt"
    "errors"
    "strings"
    "strconv"
)

// Add adds two numbers
func Add(a, b int) int {
    return a + b
}

// Subtract subtracts two numbers
func Subtract(a, b int) int {
    return a - b
}

// Divide divides two numbers with error handling
func Divide(a, b int) (int, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}

// ValidateEmail validates an email address
func ValidateEmail(email string) bool {
    if email == "" {
        return false
    }
    return strings.Contains(email, "@") && strings.Contains(email, ".")
}

// ProcessString processes a string with various conditions
func ProcessString(input string) string {
    if len(input) == 0 {
        return "empty"
    }
    if len(input) > 100 {
        return "too_long"
    }
    if strings.Contains(input, " ") {
        return "has_spaces"
    }
    return "normal"
}

// ConvertToInt converts string to integer
func ConvertToInt(s string) (int, error) {
    if s == "" {
        return 0, errors.New("empty string")
    }
    return strconv.Atoi(s)
}

// FormatName formats a person's name
func FormatName(firstName, lastName string) string {
    if firstName == "" && lastName == "" {
        return "Unknown"
    }
    if firstName == "" {
        return lastName
    }
    if lastName == "" {
        return firstName
    }
    return fmt.Sprintf("%s %s", firstName, lastName)
}
