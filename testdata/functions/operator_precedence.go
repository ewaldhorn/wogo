package calc

func AddMulPrecedence(a int32, b int32, c int32) int32 {
	return a + b*c
}

func MulAddPrecedence(a int32, b int32, c int32) int32 {
	return a*b + c
}

func SubDivPrecedence(a int32, b int32, c int32) int32 {
	return a - b/c
}

func ParenOverride(a int32, b int32, c int32) int32 {
	return (a + b) * c
}

func NestedParens(a int32, b int32, c int32, d int32) int32 {
	return ((a + b) * c) - d
}

func MixedParens(a int32, b int32, c int32, d int32) int32 {
	return (a + b) * (c - d)
}
