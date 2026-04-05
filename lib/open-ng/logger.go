package main

import (
	"log"
	"os"
)

var (
	logger	= log.New(os.Stdout, "[Portable OpenURI] [Log]", 0)
	warn	= log.New(os.Stdout, "[Portable OpenURI] [Warn]", 0)
)