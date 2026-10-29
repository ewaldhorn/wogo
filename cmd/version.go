package cmd

import (
	"fmt"
	"wogo/constants"
)

func PrintVersion() {
	fmt.Printf("The version is %s", constants.WOGO_VERSION)
}
