package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"wogo/cmd"
	"wogo/constants"
)

// ------------------------------------------------------------------------------------------------
func parseFlags() {
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Print the application version and exit")
	flag.BoolVar(&showVersion, "v", false, "Print the application version and exit")

	var showHelp bool
	flag.BoolVar(&showHelp, "help", false, "Print the application help and exit")
	flag.BoolVar(&showHelp, "h", false, "Print the application help and exit")

	flag.Usage = func() {
		printUsage()
	}

	flag.Parse()

	if showVersion || flagIs("version") {
		cmd.PrintVersion()
		exitOK()
	}

	if showHelp || flagIs("help") {
		cmd.PrintHelp()
		exitOK()
	}

}

// ------------------------------------------------------------------------------------------------
func printUsage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  -v, --version  Print the application version\n")
	fmt.Fprintf(flag.CommandLine.Output(), "  -h, --help     Print the application help\n")
}

// ------------------------------------------------------------------------------------------------
func flagIs(expected string) bool {
	return flag.NArg() > 0 && strings.EqualFold(flag.Arg(0), expected)
}

// ------------------------------------------------------------------------------------------------
func validateEnoughArguments() {
	if flag.NArg() < 2 {
		printPositionalUsage()
		exitNotEnoughArguments()
	}
}

// ------------------------------------------------------------------------------------------------
func printPositionalUsage() {
	fmt.Println("Usage: wogo input_src output_file [output_wasm_file]")
	fmt.Println("Example: wogo mycoolfile.go bundle.wat")
	fmt.Println("Example: wogo mycoolfile.go bundle.wat bundle.wasm")
}

// ------------------------------------------------------------------------------------------------
// Assume the correct number of arguments mean we have what we need
func getArguments() (string, string, string) {
	input, target, wasm, err := splitArguments(flag.Args())
	if err != nil {
		fmt.Println(err)
		printPositionalUsage()
		exitNotEnoughArguments()
	}

	return input, target, wasm
}

// ------------------------------------------------------------------------------------------------
// splitArguments maps positional arguments to the input, target, and optional wasm file slots.
// Flags must come before positionals: the flag package stops parsing at the first positional, so a
// flag placed after positionals is rejected with a usage message.
func splitArguments(args []string) (string, string, string, error) {
	for _, a := range args {
		if len(a) > 1 && strings.HasPrefix(a, "-") {
			return "", "", "", fmt.Errorf("unexpected flag %q after positional arguments (flags must come first)", a)
		}
	}
	if len(args) > 2 {
		return args[0], args[1], args[2], nil
	}
	return args[0], args[1], "", nil
}

// ------------------------------------------------------------------------------------------------
func validateFileExists(source string) {
	_, err := os.Stat(source)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Printf("File \"%v\" not found!\n", source)
		exitInputFileNotFound()
	}
}

// ------------------------------------------------------------------------------------------------
func exitOK() {
	os.Exit(constants.EXIT_CODE_OK)
}

// ------------------------------------------------------------------------------------------------
func exitNotEnoughArguments() {
	os.Exit(constants.EXIT_CODE_NOT_ENOUGH_ARGUMENTS)
}

// ------------------------------------------------------------------------------------------------
func exitInputFileNotFound() {
	os.Exit(constants.EXIT_CODE_INPUT_FILE_NOT_FOUND)
}

// ------------------------------------------------------------------------------------------------
func exitUnableToReadInputFile() {
	os.Exit(constants.EXIT_CODE_ERROR_READING_INPUT_FILE)
}

// ------------------------------------------------------------------------------------------------
func exitUnableToWritingOutputFile() {
	os.Exit(constants.EXIT_CODE_ERROR_WRITING_OUTPUT_FILE)
}
