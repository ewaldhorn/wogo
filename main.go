package main

import (
	"fmt"
	"os"
	"wogo/internal/compiler"
	"wogo/internal/parser"
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

	p := &parser.Parser{}
	writeTargetFile(targetFile, p.Parse(src))

	// now that we have parsed the WAT without error, we can compile it to WASM
	if wasmFile != "" {
		c := compiler.NewCompiler()
		wasmBytes, err := c.Compile(src)
		if err != nil {
			fmt.Printf("Error compiling WASM: %v\n", err)
			exitUnableToWritingOutputFile()
		}
		writeTargetBytes(wasmFile, wasmBytes)
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
// TODO: Better error handling and report. OK for now while I'm still figuring things out.
func writeTargetFile(tgt, data string) {
	writeTargetBytes(tgt, []byte(data))
}

// ------------------------------------------------------------------------------------------------
func writeTargetBytes(tgt string, data []byte) {
	err := os.WriteFile(tgt, data, 0644)
	if err != nil {
		fmt.Println("Error writing output file:", err)
		exitUnableToWritingOutputFile()
	}
}
