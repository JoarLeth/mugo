package testutil

import "math"

// AreDifferent(x, y) bekaves like x != y, but sees NaN() as equal to NaN()
func AreDifferent(x float64, y float64) bool {
	if math.IsNaN(x) && math.IsNaN(y) {
		return false
	}

	return x != y
}
