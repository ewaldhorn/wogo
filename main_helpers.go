package main

import (
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
func exitOK() {
	os.Exit(constants.EXIT_CODE_OK)
}
