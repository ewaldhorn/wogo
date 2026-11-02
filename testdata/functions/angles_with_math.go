package angles

import "math"

func DegreesToRads(deg float64) float64 {
	return deg * (math.Pi / 180.0)
}

func RadsToDegrees(rad float64) float64 {
	return rad * (180.0 / math.Pi)
}
