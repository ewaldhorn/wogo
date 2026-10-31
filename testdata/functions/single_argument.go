package calc

func Identity(a int32) int32 {
	return a
}

func Negate(a int32) int32 {
	return -a
}

func Double(a int32) int32 {
	return a + a
}

func Square(a int32) int32 {
	return a * a
}

func Cube(a int32) int32 {
	return a * a * a
}

func IsFalse() bool {
	return false
}
