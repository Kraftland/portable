package main

import (
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
	var trueColor bool
	if os.Getenv("COLORTERM") == "truecolor" {
		// See https://en.wikipedia.org/wiki/ANSI_escape_code#Unix_environment_variables_relating_to_color_support
		trueColor = true
	}
	if len(os.Getenv("NO_COLOR")) > 0 {
		trueColor = false
		pecho("debug", "Disabled coloured output in response to NO_COLOR")
	}
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
	var debugLogger *log.Logger
	var critLogger *log.Logger
	var warnLogger *log.Logger
	var infoLogger *log.Logger

	if trueColor {
		debugLogger = log.New(
			os.Stdout,
			"\033[0m" + "\033[38;2;125;241;118m" + "[Debug] " + "\033[0m",
			0,
		)
		critLogger = log.New(
			os.Stderr,
			"\033[0m" + "\033[38;2;255;0;0m" + "[Critical] " + "\033[0m",
			0,
		)
		warnLogger = log.New(
			os.Stderr,
			"\033[0m" + "\033[38;2;255;209;59m" + "[Warn] " + "\033[0m",
			0,
		)
		infoLogger = log.New(
			os.Stderr,
			"\033[0m" + "\033[38;2;119;222;250m" + "[Info] " + "\033[0m",
			0,
		)
	} else {
		debugLogger = log.New(
			os.Stdout,
			"\033[0m" + "[Debug] " + "\033[0m",
			0,
		)
		critLogger = log.New(
			os.Stderr,
			"\033[0m" + "[Critical] " + "\033[0m",
			0,
		)
		warnLogger = log.New(
			os.Stderr,
			"\033[0m" + "[Warn] " + "\033[0m",
			0,
		)
		infoLogger = log.New(
			os.Stderr,
			"\033[0m" + "[Info] " + "\033[0m",
			0,
		)
	}

	for {
		chanRes := <- pechoChan
		switch chanRes.level {
			case "debug":
				if internalLoggingLevel <= 1 {
					debugLogger.Println(chanRes.msg)
				}
			case "info":
				if internalLoggingLevel <= 2 {
					infoLogger.Println(chanRes.msg)
				}
			case "warn":
				warnLogger.Println(chanRes.msg)
			case "crit":
				critLogger.Println(chanRes.msg)
				select {
					case stopSig <- 1:
					default:
						critLogger.Fatalln(
							"This critical error happened before stopper initialisation",
						)
				}
			default:
				pecho("crit", "Unknown message level for", chanRes.msg)
		}
	}
}