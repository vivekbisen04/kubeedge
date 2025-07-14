package math

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
	testCases := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"Positive numbers", 1, 2, 3},
		{"Negative numbers", -1, -2, -3},
		{"Zero and positive", 0, 5, 5},
		{"Positive and zero", 5, 0, 5},
		{"Large numbers", 1000000, 2000000, 3000000},
		{"Zero and zero",0,0,0},
		{"Negative and positive", -5, 10, 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, Add(tc.a, tc.b))
		})
	}
}


func TestSubtract(t *testing.T) {
	testCases := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"Positive numbers", 5, 2, 3},
		{"Negative numbers", -5, -2, -3},
		{"Zero and positive", 0, 5, -5},
		{"Positive and zero", 5, 0, 5},
		{"Large numbers", 1000000, 200000, 800000},
		{"Zero and zero",0,0,0},
		{"Negative and positive", -5, 10, -15},
		{"Positive and negative", 5, -10, 15},

	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, Subtract(tc.a, tc.b))
		})
	}
}

func TestMultiply(t *testing.T) {
	testCases := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"Positive numbers", 5, 2, 10},
		{"Negative numbers", -5, -2, 10},
		{"Zero and positive", 0, 5, 0},
		{"Positive and zero", 5, 0, 0},
		{"Large numbers", 1000, 2000, 2000000},
		{"Zero and zero",0,0,0},
		{"Negative and positive", -5, 10, -50},
		{"Positive and negative", 5, -10, -50},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, Multiply(tc.a, tc.b))
		})
	}
}

func TestIsPositive(t *testing.T) {
	testCases := []struct {
		name     string
		n        int
		expected bool
	}{
		{"Positive number", 5, true},
		{"Negative number", -5, false},
		{"Zero", 0, false},
		{"Large number", 1000000, true},
		{"Small number",1, true},
		{"Negative large number", -1000000, false},

	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, IsPositive(tc.n))
		})
	}
}