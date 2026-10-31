package arguments

func IsFalse() bool {
	return false
}

func IsTrue() bool {
	return true
}

func IsNotTrue() bool {
	return !true
}

func IsNotFalse() bool {
	return !false
}

// double NOT returns the original value
func IsActuallyFalse() bool {
	return !!false
}

func IsActuallyTrue() bool {
	return !!true
}

// an odd number of NOTS returns the opposite of the start value
func FiveNots() bool {
	return !!!!!true
}

// an even number of NOTS returns the start value
func SixNots() bool {
	return !!!!!!true
}

// odd again so expecting the opposite
func FifteenNots() bool {
	return !!!!!!!!!!!!!!!true
}
