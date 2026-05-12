package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func printHelp() {
	var builder strings.Builder
	builder.WriteString(
		"This is Portable, a fast, private, modern sandbox designed for desktop Linux. \n",
	)

	file, err := os.Open(
		"/usr/share/doc/portable/General/01-cli-arguments.md",
	)
	if err != nil {
		pecho("warn", "Could not open help page:", err)
	} else {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			builder.WriteString(scanner.Text())
			builder.WriteString("\n")
		}
	}

	builder.WriteString("This Portable install comes with super golden power\n")
	fmt.Println(builder.String())
}