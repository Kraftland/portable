package main

import (
	"os"

	"golang.org/x/term"
)

func setRawConsole() {
	oldState, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		pecho("warn", "Could not get console state:", err)
		return
	}

	stopFuncChan <- func() {
		err := term.Restore(int(os.Stdin.Fd()), oldState)
		if err != nil {
			pecho("warn", "Could not restore console state:", err)
		}
	}
}