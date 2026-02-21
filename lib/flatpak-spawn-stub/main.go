package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"os/exec"
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
	cmdSlice := os.Args[1:]
	log.Println("Portable flatpak-spawn stub version: " + strconv.FormatFloat(version, 'g', -1, 64))
	log.Println("Full cmdline: " + strings.Join(cmdSlice, ", "))

	var knownArgs int
	var appTgt []string
	if len(cmdSlice) > 1 {
		for idx, flag := range cmdSlice {
			if strings.HasPrefix(flag, "--") == false {
				log.Println("Appending application arguments", cmdSlice[idx:])
				appTgt = append(appTgt, cmdSlice[idx:]...)
				break
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
							log.Println("Could not open file descriptor: " + err.Error())
							//os.Exit(2)
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

	allFlagCnt := len(cmdSlice)
	log.Println("Resolution of cmdline finished: " + strconv.Itoa(knownArgs) + " of " + strconv.Itoa(allFlagCnt) + " readable")

	fds := []uintptr{0, 1, 2}

	fds[fdFwd.Fd() + 1] = fdFwd.Fd()

	attrs := &syscall.ProcAttr{
		Dir:		chDir,
		Env:		append(os.Environ(), envAdd...),
		Files:		fds,
	}
	if clearEnv {
		attrs.Env = []string{}
	}

	resolvBinPath, err := exec.LookPath(appTgt[0])
	if err != nil {
		log.Fatalln("Could not look up executable: " + err.Error())
	}

	pid, err := syscall.ForkExec(resolvBinPath, appTgt, attrs)
	if err != nil {
		log.Fatalln("Could not fork exec: " + err.Error())
	}

	log.Println("Started underlying process " + strconv.Itoa(pid) + " :", appTgt)

	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGILL, syscall.SIGILL, syscall.SIGINT)
	go terminateWatcher(sigChan)
	//<- waitChan
}