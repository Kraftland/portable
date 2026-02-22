package main

import (
	"os"
	"log"
	"strconv"
	"strings"
	"syscall"
	"os/exec"
	"time"
)

func main () {
	log.Flags()
	cmdSlice := os.Args[1:]
	log.Println("Full cmdline: " + strings.Join(cmdSlice, ", "))

	var knownArgs int
	var appTgt []string
	if len(cmdSlice) > 1 {
		for idx, flag := range cmdSlice {
			if strings.HasPrefix(flag, "-") == false {
				log.Println("Appending application arguments", cmdSlice[idx:])
				appTgt = append(appTgt, cmdSlice[idx:]...)
				break
			}
			switch flag {
				default:
					log.Println("Unknown argument: " + flag)
					continue
			}
			knownArgs++
		}
	}

	allFlagCnt := len(cmdSlice)
	log.Println("Resolution of cmdline finished: " + strconv.Itoa(knownArgs) + " of " + strconv.Itoa(allFlagCnt) + " readable")
	resolvBinPath, err := exec.LookPath(appTgt[0])
	if err != nil {
		log.Println("Could not look up executable: " + err.Error())
		_, err = os.Stat(appTgt[0])
		if err != nil {
			resolvBinPath = "/usr/bin/" + appTgt[0]
		} else {
			resolvBinPath = appTgt[0]
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalln("Could not get working directory: " + err.Error())
	}
	attrs := &syscall.ProcAttr{
		Dir:		wd,
		Env:		os.Environ(),
		Sys:		&syscall.SysProcAttr{
					Pdeathsig:		syscall.SIGKILL,
		},
	}
	pid, err := syscall.ForkExec(resolvBinPath, appTgt, attrs)
	log.Println("Started process", pid)
	var wstat syscall.WaitStatus
	for {
		wpid, err := syscall.Wait4(pid, &wstat, 0, nil)
		if err != nil {
			switch err {
				case syscall.EINTR:
					time.Sleep(1 * time.Second)
					continue
				default:
					log.Fatalln("Could not watch PID:", err.Error())
			}
		}
		if pid == wpid {
			break
		}
	}
}