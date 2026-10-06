package utils

import "math"

// RupeesToPaise converts a float64 rupee amount to int64 paise (minor units).
// Uses math.Round to avoid floating-point truncation issues.
//
// Example:
//
//	RupeesToPaise(5000.50) => 500050
//	RupeesToPaise(100.00)  => 10000
func RupeesToPaise(rupees float64) int64 {
	return int64(math.Round(rupees * 100))
}

// PaiseToRupees converts int64 paise to float64 rupees.
// Use only at API/DTO boundaries — never for internal money arithmetic.
//
// Example:
//
//	PaiseToRupees(500050) => 5000.50
func PaiseToRupees(paise int64) float64 {
	return float64(paise) / 100.0
}
