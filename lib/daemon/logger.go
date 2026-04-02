package main

import (
	"fmt"
	"os"
)

var (
	pechoChan		= make(chan []string, 128)
)

func pecho(level string, message string) {
	var msgSlice = []string{
		level,
		message,
	}
	pechoChan <- msgSlice
}


func pechoWorker() {
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
	pechoChan <- []string{
		"debug",
		"Initialized logging daemon",
	}
	for {
		chanRes := <- pechoChan
		switch chanRes[0] {
			case "debug":
				if internalLoggingLevel <= 1 {
					fmt.Println("[Debug] ", chanRes[1])
				}
			case "info":
				if internalLoggingLevel <= 2 {
					fmt.Println("[Info] ", chanRes[1])
				}
			case "warn":
				fmt.Println("[Warn] ", chanRes[1])
			case "crit":
				fmt.Println("[Critical] ", chanRes[1])
				stopApp()
				panic("A critical error has happened")
			default:
				fmt.Println("[Undefined] ", chanRes[1])
		}
	}
}