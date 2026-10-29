package cmd

import (
	"fmt"
	"wogo/constants"
)

// ------------------------------------------------------------------------------------------------
func PrintVersion() {
	fmt.Printf("Wogo version %s.\n", constants.WOGO_VERSION)
}
