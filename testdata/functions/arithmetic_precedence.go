package calc

func Precedence(a int32, b int32, c int32) int32 {
	return a + b*c
}

func Grouped(a int32, b int32, c int32) int32 {
	return (a + b) * c
}

func AddConstant(a int32) int32 {
	return a + 10
}
