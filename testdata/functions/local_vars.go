package local_vars

func ShortVar(x int32) int32 {
	y := x + 10
	return y
}

func Reassign(x int32) int32 {
	y := x
	y = y * 2
	return y
}

func CompoundOps(x int32) int32 {
	y := x
	y += 10
	y -= 2
	y *= 3
	y /= 2
	y %= 7
	return y
}

func VarZero() int32 {
	var x int32
	return x
}

func VarInit(x int32) int32 {
	var y int32 = x + 5
	return y
}

func MultipleLocals(a int32, b int32) int32 {
	x := a * 2
	y := b * 3
	x += y
	return x
}

func FloatLocal(a float64) float64 {
	step := 1.5
	res := a + step
	res *= 2.0
	return res
}

func MutateParam(x int32) int32 {
	x += 5
	return x
}

func MixedShortVar(x int32) int32 {
	y := x
	y, z := y+1, 10
	return y + z
}

func Swap(a int32, b int32) int32 {
	a, b = b, a
	return a
}

func DupAssign() int32 {
	x := 0
	x, x = 1, 2
	return x
}
