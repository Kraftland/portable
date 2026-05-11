package main

import (
	"fmt"
	"log"
	"os"
)

var (
	pechoChan		= make(chan pechoMsg, 128)
)

type pechoMsg struct {
	level		string
	msg		[]any
}

func pecho(level string, message ...any) {
	pechoChan <- pechoMsg {
		level:	level,
		msg:	message,
	}
}


func pechoWorker(stopSig chan int) {
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
	var debugLogger = log.New(os.Stdout, "[Debug] ", 0)
	var critLogger = log.New(os.Stderr, "[Critical] ", 0)
	var warnLogger = log.New(os.Stderr, "[Warn] ", 0)
	for {
		chanRes := <- pechoChan
		switch chanRes.level {
			case "debug":
				if internalLoggingLevel <= 1 {
					debugLogger.Println(chanRes.msg)
				}
			case "info":
				if internalLoggingLevel <= 2 {
					fmt.Println("[Info] ", chanRes.msg)
				}
			case "warn":
				warnLogger.Println(chanRes.msg)
			case "crit":
				critLogger.Println(chanRes.msg)
				select {
					case stopSig <- 1:
					default:
						critLogger.Fatalln(
							"This critical error happened before stopper has initialised",
						)
				}
			default:
				pecho("crit", "Unknown message level for", chanRes.msg)
		}
	}
}