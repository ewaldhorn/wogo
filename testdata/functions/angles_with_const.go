package angles

const PI = 3.141592653589793

func DegreesToRads(deg float64) float64 {
	return deg * (PI / 180.0)
}

func RadsToDegrees(rad float64) float64 {
	return rad * (180.0 / PI)
}

// Shadow the global PI with a clearly wrong number
func ShadowPI() float64 {
	const PI = 3.15
	return PI
}

func Circumference(radius float64) float64 {
	const LocalPi = 3.141592653589793
	return 2.0 * LocalPi * radius
}
