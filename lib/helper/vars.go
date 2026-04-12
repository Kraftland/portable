package main

import (
	"log"
	"os"
	"sync"
)

var (
	debug = log.New(os.Stdout, "[Portable] [Init] ", 0)
	warn = log.New(os.Stderr, "[Portable] [Init] [Warning] ", 0)

	// Origin -> Dest, must lock to prevent races
	fileSubstitutionMap	= make(map[string]string)
	fileMapLock		sync.RWMutex
)