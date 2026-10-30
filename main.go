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
	inputSource, targetFile, wasmFile := getArguments()

	fmt.Printf("Input source: %s\n", inputSource)
	fmt.Printf("Target file : %s\n", targetFile)
	if wasmFile != "" {
		fmt.Printf("WASM file   : %s\n", wasmFile)
	}

	validateFileExists(inputSource)
	src := readInputSourceFile(inputSource)

	fmt.Println("Going to try to build:")
	fmt.Println()

	parser := &compiler.Parser{}
	writeTargetFile(targetFile, parser.Parse(src))

	// now that we have a WAT file, we can try to compile it to WASM
	// this is highly experimental and stupid right now, but hey...
	if wasmFile != "" {
		fmt.Println("WASM file is TODO")
	}
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

// ------------------------------------------------------------------------------------------------
func writeTargetFile(tgt, data string) {
	err := os.WriteFile(tgt, []byte(data), 0644)
	if err != nil {
		fmt.Println("Error writing output file:", err)
		exitUnableToWritingOutputFile()
	}
}
