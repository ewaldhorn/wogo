package main

import (
	"fmt"
	"wogo/internal/compiler"
)

// ------------------------------------------------------------------------------------------------
func main() {
	parseFlags()
	validateEnoughArguments()
	inputSource, targetFile := getArguments()

	fmt.Printf("Input source: %s\n", inputSource)
	fmt.Printf("Target file : %s\n", targetFile)

	validateFileExists(inputSource)

	fmt.Println("Going to try to build:")
	fmt.Println()

	src := `
	package calc

	func Compute(a int32, b int32) int32 {
		return (a + 10) * (b - 2)
	}
	`

	compiler := &compiler.Compiler{}
	compiler.Compile(src)
}
