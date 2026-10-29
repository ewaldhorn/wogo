package main

import (
	"fmt"
	"os"
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
	src := readInputSourceFile(inputSource)

	fmt.Println("Going to try to build:")
	fmt.Println()

	compiler := &compiler.Compiler{}
	compiler.Compile(src)
}

// ------------------------------------------------------------------------------------------------
func readInputSourceFile(src string) string {
	content, err := os.ReadFile(src)
	if err != nil {
		fmt.Printf("Error reading file content from \"%s\": %v\n", src, err.Error())
		exitUnableToReadInputFile()
	}

	return string(content)
}
