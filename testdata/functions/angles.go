package angles

func DegreesToRads(deg float64) float64 {
	return deg * (3.141592653589793 / 180.0)
}

func RadsToDegrees(rad float64) float64 {
	return rad * (180.0 / 3.141592653589793)
}

