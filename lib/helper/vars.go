package main

import (
	"log"
	"os"
)

var (
	debug = log.New(os.Stdout, "[Portable] [Init] ", 0)
	warn = log.New(os.Stderr, "[Portable] [Init] [Warning] ", 0)
)