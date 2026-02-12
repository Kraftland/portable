package main

import (
	"fmt"
	"os"
	"strings"
)


func main () {
	rawCmdArgs := os.Args
	fmt.Println("Received command line open request: " + strings.Join(rawCmdArgs, ", "))
}