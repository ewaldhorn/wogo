package scoping

func SequentialLoops(n int32) int32 {
	sum := 0
	for i := 0; i < n; i++ {
		sum += i
	}
	for i := 0; i < n; i++ {
		sum += i * 2
	}
	return sum
}

func ShadowOuterVar(a int32) int32 {
	x := a + 1
	{
		x := x * 2
		a += x
	}
	return x + a
}

func ShadowParam(a int32) int32 {
	res := 0
	{
		a := a + 10
		res = a
	}
	return res + a
}

func ShadowDifferentType(a int32) float64 {
	x := a
	res := 0.0
	{
		x := 2.5
		res = x
	}
	if x > 0 {
		res += 1.0
	}
	return res
}

func IfElseInitScoping(a int32) int32 {
	out := 0
	if x := a * 2; x > 10 {
		x := x + 1
		out = x
	} else {
		out = x
	}
	if x := a + 100; x > 105 {
		out += x
	}
	return out
}

func ForBodyShadowsLoopVar(n int32) int32 {
	sum := 0
	for i := 0; i < n; i++ {
		i := 100
		sum += i
	}
	return sum
}

func LoopVarZeroInit(n int32) int32 {
	total := 0
	for i := 0; i < n; i++ {
		var acc int32
		acc += i
		total += acc
	}
	return total
}

func ScopedConstants(a int32) int32 {
	const k = 10
	res := k
	{
		const k = k + 5
		res += k
	}
	{
		k := a
		res += k
	}
	return res + k
}

func NameCollisionAvoidance(a int32) int32 {
	x := a
	x_1 := 100
	{
		x := 7
		x_1 += x
	}
	return x + x_1
}

func BlankIdentifiers(_ int32, a int32) int32 {
	_ = a + 1
	_, _, x := 1, 2.5, a*2
	var _ int32 = 99
	const _ = 42
	return x
}
