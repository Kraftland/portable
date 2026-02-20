package main

import (
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"os/signal"
	"syscall"
)

const (
	version		float64		=	0.1
)

func terminateWatcher(term bool, sigChan chan os.Signal) {
	sig := <- sigChan
	log.Println("Got signal: " + sig.String() + ", terminating flatpak-spawn stub")

	if term {

	}
	os.Exit(0)
}

func main() {
	var sigChan = make(chan os.Signal, 1)
	cmdSlice := os.Args
	log.Println("Portable flatpak-spawn stub version: " + strconv.FormatFloat(version, 'g', -1, 64))
	log.Println("Full cmdline: " + strings.Join(cmdSlice, ", "))

	var knownArgs int
	var appTgt []string
	if len(cmdSlice) > 1 {
		for _, flag := range cmdSlice[1:] {
			if strings.HasPrefix(flag, "--") == false {
				appTgt = append(appTgt, flag)
				knownArgs++
				continue
			}
			switch flag {
				case "--sandbox":
					log.Println("Ignoring --sandbox because already in sandbox")
				case "--watch-bus":
					log.Println("Watching termination")
				default:
					log.Println("Unknown flag: " + flag)
					continue
			}
			knownArgs++
		}
	}

	allFlagCnt := len(cmdSlice) - 1
	log.Println("Resolution of cmdline finished: " + strconv.Itoa(knownArgs) + " of " + strconv.Itoa(allFlagCnt) + " readable")

	cmd := exec.Command(appTgt[0], appTgt[1:]...)

	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGILL, syscall.SIGILL, syscall.SIGINT)
}