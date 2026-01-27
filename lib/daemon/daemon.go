package main

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	version float32 = 0.1
)

var (
	internalLoggingLevel int
	appID string
	runtimeDir string
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
			fmt.Println("[Critical] ", message)
			panic("A critical error has happened")
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
	runtimeDir = os.Getenv("XDG_RUNTIME_DIR")
	if len(runtimeDir) == 0 {
		pecho("warn", "XDG_RUNTIME_DIR not set")
	} else {
		var runtimeDebugMsg string = "XDG_RUNTIME_DIR set to: " + runtimeDir
		pecho("debug", runtimeDebugMsg)
		runtimeDirInfo, errRuntimeDir := os.Stat(runtimeDir)
		var errRuntimeDirPrinted string = "Could not determine the status of XDG Runtime Directory "
		if errRuntimeDir != nil {
			println(errRuntimeDir)
			pecho("crit", errRuntimeDirPrinted)
		}
		if runtimeDirInfo.IsDir() == false {
			pecho("crit", "XDG_RUNTIME_DIR is not a directory")
		}
	}
	appID = os.Getenv("appID")
	if len(appID) == 0 {
		pecho("crit", "Application ID unknown")
	}
}

func startApp() {
	sdExec := exec.Command("xargs", "-0")
	sdExec.Stderr = os.Stderr
	argFile, argOpenErr := os.Open(runtimeDir + "/portable/" + appID + "/bwrapArgs")
	sdExec.Stdin = argFile
	if argOpenErr != nil {
		pecho("crit", "Could not read file: " + argOpenErr.Error())
	}
	fmt.Println("Executing ", sdExec)
	sdExecErr := sdExec.Run()
	if sdExecErr != nil {
		fmt.Println(sdExecErr)
		pecho("crit", "Unable to start systemd-run")
	}
}

func main() {
	fmt.Println("Portable daemon", version, "starting")
	getVariables()
	startApp()
}