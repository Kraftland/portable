package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	version		float64		=	0.1
)

var (
	clearEnv			bool
	envAdd				[]string
	fdFwd				*os.File
	proc				*os.Process
	term				bool
	chDir				string
	waitChan			= make(chan int, 1)
)

func fdWatcher(sigChan chan os.Signal) {

	for {
		time.Sleep(5 * time.Second)
		_, err := fdFwd.Stat()
		if err != nil {
			log.Println("Exiting on fd read err: " + err.Error())
			break
		}
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
	close(waitChan)
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
						fdNums := strings.TrimPrefix(flag, "--forward-fd=")
						openFd, err := os.Open("/proc/self/fd/" + fdNums)
						if err != nil {
							log.Fatalln("Could not open file descriptor: " + err.Error())
						}
						fdFwd = openFd
						//fdFwd = os.NewFile(uintptr(fdNums), "passedFd")
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
		cycleCount := int(fdFwd.Fd()) - 3
		currCycle := 0
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
	<- waitChan
}