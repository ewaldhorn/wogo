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
	return flag.NArg() > 00 && strings.EqualFold(flag.Arg(0), expected)
}

// ------------------------------------------------------------------------------------------------
func validateEnoughArguments() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: wogo input_src output_file")
		fmt.Println("Example: wogo mycoolfile.go bundle.wat")
		exitNotEnoughArguments()
	}
}

// ------------------------------------------------------------------------------------------------
// Assume the correct number of arguments mean we have what we need
func getArguments() (string, string) {
	return os.Args[1], os.Args[2]
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
