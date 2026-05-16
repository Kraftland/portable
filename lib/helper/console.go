package main

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

func setRawConsole() {
	oldState, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		warn.Println("Could not get console state:", err)
		return
	}

	go func (oldState *term.State) {
		var sigChan = make(chan os.Signal, 1)
		signal.Notify(sigChan)
		for sig := range sigChan {
			if sig == syscall.SIGTERM {
				err := term.Restore(int(os.Stdin.Fd()), oldState)
				if err != nil {
					warn.Println("Could not restore console state:", err)
				}
				break
			}
		}
	} (oldState)
}