package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
)

const (
	version float32 = 0.1
)

var (
	internalLoggingLevel int
)

func pecho(level string, message string) {
	switch level {
		case "debug":
			if internalLoggingLevel <= 1 {
				fmt.Println("[Debug] ", message)
			}
		case "info":
			if internalLoggingLevel <= 2 {
				fmt.Println("[Info] ", message)
			}
		case "warn":
			fmt.Println("[Warn] ", message)
		case "crit":
			log.Panicln("[Critical] ", message)
		default:
			fmt.Println("[Undefined] ", message)
	}
}

func getVariables() {
	var externalLoggingLevel = os.Getenv("PORTABLE_LOGGING")
	switch externalLoggingLevel {
		case "debug":
			internalLoggingLevel = 1
		case "info":
			internalLoggingLevel = 2
		case "warn":
			internalLoggingLevel = 3
		default:
			internalLoggingLevel = 3
	}
	var runtimeDir string = os.Getenv("XDG_RUNTIME_DIR")
	if len(runtimeDir) == 0 {
		pecho("warn", "XDG_RUNTIME_DIR not set")
	} else {
		var runtimeDebugMsg string = "XDG_RUNTIME_DIR set to: " + runtimeDir
		pecho("debug", runtimeDebugMsg)
		runtimeDirInfo := fs.FileInfo(
		var runtimeDirExists bool = os.Open()
	}
}

func startApp() {
	//var sdArguments = os.Open("")
}

func main() {
	fmt.Println("Portable daemon", version, "starting")
	getVariables()
}