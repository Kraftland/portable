package main

import (
	"log"
	"os"
	"sync"
)

var (
	debug = log.New(os.Stdout, "\033[0m" + "\033[38;2;125;241;118m" + "[Init]	" + "\033[0m", 0)
	warn = log.New(os.Stderr, "\033[0m" + "\033[38;2;255;209;59m" + "[Init]	" + "\033[0m", 0)

	// Origin -> Dest, must lock to prevent races
	fileSubstitutionMap	= make(map[string]string)
	fileMapLock		sync.RWMutex
)