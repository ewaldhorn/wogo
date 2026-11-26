package if_else

func Max(a, b int32) int32 {
	if a > b {
		return a
	} else {
		return b
	}
}

func Abs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

func Clamp(x, min, max int32) int32 {
	if x < min {
		return min
	} else if x > max {
		return max
	} else {
		return x
	}
}

func WithInit(a int32) int32 {
	if x := a * 2; x > 10 {
		return x
	} else {
		return 0
	}
}

func NestedIf(a, b int32) int32 {
	if a > 0 {
		if b > 0 {
			return 1
		} else {
			return 2
		}
	} else {
		if b > 0 {
			return 3
		} else {
			return 4
		}
	}
}
