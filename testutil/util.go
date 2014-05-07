package testutil

import (
	"math"
	"math/rand"
)

// AreDifferent(x, y) bekaves like x != y, but sees NaN() as equal to NaN()
func AreDifferent(x float64, y float64) bool {
	if math.IsNaN(x) && math.IsNaN(y) {
		return false
	}

	if x == y && math.Signbit(x) != math.Signbit(y) { // 0 != -0
		return true
	}

	return x != y
}

var Float64SpecialCases = []float64{0, math.Copysign(0, -1), math.Inf(1), math.Inf(-1), math.NaN()}

func RandomFloat64SpecialCase(r *rand.Rand) float64 {
	n := r.Intn(len(Float64SpecialCases))
	return Float64SpecialCases[n]
}
