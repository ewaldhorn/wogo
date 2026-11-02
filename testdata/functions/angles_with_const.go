package angles

const PI = 3.141592653589793

func DegreesToRads(deg float64) float64 {
	return deg * (PI / 180.0)
}

func RadsToDegrees(rad float64) float64 {
	return rad * (180.0 / PI)
}
