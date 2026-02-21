package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"golang.org/x/sys/unix"
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
	chDir				string
)

func fdWatcher(sigChan chan os.Signal) {
	pfd := []unix.PollFd{
		{
			Fd:		int32(fdFwd.Fd()),
			Events:		unix.POLLHUP | unix.POLLERR,
		},
	}

	_, err := unix.Poll(pfd, -1)
	if err != nil {
		log.Fatalln("Could not poll fd: " + err.Error())
	}

	sigChan <- syscall.SIGTERM

}

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
	chDir, _ = os.Getwd()
	cmdSlice := os.Args
	log.Println("Portable flatpak-spawn stub version: " + strconv.FormatFloat(version, 'g', -1, 64))
	log.Println("Full cmdline: " + strings.Join(cmdSlice, ", "))

	var knownArgs int
	var appTgt []string
	var selfArgEnd bool
	if len(cmdSlice) > 1 {
		for _, flag := range cmdSlice[1:] {
			if strings.HasPrefix(flag, "--") == false || selfArgEnd {
				selfArgEnd = true
				log.Println("Appending " + flag + " to application arguments")
				appTgt = append(appTgt, flag)
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
					} else if strings.HasPrefix(flag, "--directory=") {
						chDir = strings.TrimPrefix(flag, "--directory=")
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
			currCycle++
		}
		go fdWatcher(sigChan)
	}

	log.Println("Started underlying process with: " + strings.Join(cmd.Args, " "))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
	proc = cmd.Process

	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGILL, syscall.SIGILL, syscall.SIGINT)
	go terminateWatcher(sigChan)


	cmd.Wait()
}