package main

import (
	"fmt"
	"io"
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
	file, err := os.Open(src)
	if err != nil {
		fmt.Printf("Error accessing \"%s\": %v\n", src, err.Error())
		exitUnableToReadInputFile()
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("Error reading file content from \"%s\": %v\n", src, err.Error())
		exitUnableToReadInputFile()
	}

	return string(content)
}
