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

var (
	clearEnv			bool
	envAdd				[]string
	fdFwd				*os.File
	fdNum				uint
	proc				*os.Process
	term				bool
)

func terminateWatcher(sigChan chan os.Signal) {
	sig := <- sigChan
	log.Println("Got signal: " + sig.String() + ", terminating flatpak-spawn stub")

	if term {
		if proc != nil {
			proc.Kill()
		}

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
		var skipArg bool
		for _, flag := range cmdSlice[1:] {
			if skipArg == true {
				skipArg = false
				continue
			}
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
				case "--clear-env":
					log.Println("Launching with no inherited environment variables")
				default:
					if strings.HasPrefix(flag, "--forward-fd=") {
						fdNums, err := strconv.Atoi(strings.TrimPrefix(flag, "--forward-fd="))
						if err != nil {
							log.Fatalln("Failed to parse file descriptor: " + err.Error())
						}
						fdFwd = os.NewFile(uintptr(fdNums), "passedFd")
						fdNum = uint(fdNums)
					} else if strings.HasPrefix(flag, "--env=") {
						envLine := strings.TrimPrefix(flag, "--env=")
						if strings.Contains(envLine, "=") == false {
							log.Println("Invalid env: " + envLine)
							continue
						}
						envAdd = append(envAdd, envLine)
					} else {
						log.Println("Unknown flag: " + flag)
						continue
					}
			}
			knownArgs++
		}
	}

	allFlagCnt := len(cmdSlice) - 1
	log.Println("Resolution of cmdline finished: " + strconv.Itoa(knownArgs) + " of " + strconv.Itoa(allFlagCnt) + " readable")

	cmd := exec.Command(appTgt[0], appTgt[1:]...)
	if len(envAdd) > 0 {
		cmd.Env = append(os.Environ(), envAdd...)
	}
	if clearEnv {
		cmd.Env = []string{}
	}
	if fdFwd != nil {
		cycleCount := fdNum - 3
		currCycle := uint(0)
		for {
			if currCycle == cycleCount {
				cmd.ExtraFiles = append(cmd.ExtraFiles, fdFwd)
				break
			}
			cmd.ExtraFiles = append(cmd.ExtraFiles, os.Stdout)
		}

	}
	proc = cmd.Process

	cmd.Start()

	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGILL, syscall.SIGILL, syscall.SIGINT)
	go terminateWatcher(sigChan)

	cmd.Wait()
}