package comparisons

func EqualI32(a int32, b int32) bool {
	return a == b
}

func NotEqualI32(a int32, b int32) bool {
	return a != b
}

func LessThanI32(a int32, b int32) bool {
	return a < b
}

func LessOrEqualI32(a int32, b int32) bool {
	return a <= b
}

func GreaterThanI32(a int32, b int32) bool {
	return a > b
}

func GreaterOrEqualI32(a int32, b int32) bool {
	return a >= b
}

func LessThanU32(a uint32, b uint32) bool {
	return a < b
}

func GreaterThanU32(a uint32, b uint32) bool {
	return a > b
}

func EqualI64(a int64, b int64) bool {
	return a == b
}

func LessThanI64(a int64, b int64) bool {
	return a < b
}

func GreaterThanU64(a uint64, b uint64) bool {
	return a > b
}

func LessThanF64(a float64, b float64) bool {
	return a < b
}

func GreaterOrEqualF32(a float32, b float32) bool {
	return a >= b
}

func EqualBool(a bool, b bool) bool {
	return a == b
}

func NotEqualBool(a bool, b bool) bool {
	return a != b
}

func IsPositive(x int32) bool {
	return x > 0
}

func InRange(x int32, min int32, max int32) bool {
	low := x >= min
	high := x <= max
	return low == high
}
