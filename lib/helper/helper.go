package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"bufio"
	"github.com/rclone/rclone/lib/systemd"
)

var (
	startNotifier		= make(chan bool, 32767)
)

func startCounter () {
	var startedCount int = 0
	fmt.Println("Start counter init done")
	for {
		incoming := <- startNotifier
		if incoming == true {
			startedCount++
		} else {
			startedCount = startedCount - 1
		}

		systemd.UpdateStatus("Tracking processes: " + strconv.Itoa(startedCount))

		if startedCount < 1 {
			fmt.Println("All tracked processes have exited")
			const text = "terminate-now"
			fd, err := os.OpenFile("/run/startSignal", os.O_WRONLY, 0700)
			if err != nil {
				fmt.Println("Unable to open signal file: " + err.Error())
			}
			fmt.Fprint(fd, text)
			fmt.Println("Sent termination signal")
			os.Exit(0)
		}
	}
}

func executeAndWait (launchTarget string, args []string) {
	cmd := exec.Command(launchTarget, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
	startNotifier <- true
	cmd.Wait()
	startNotifier <- false
}

func auxStart (launchTarget string) {
	inotifyArgs := []string{
		"--quiet",
		"-e",
		"modify",
		"/run/startSignal",
	}
	fd, err := os.OpenFile("/run/startSignal", os.O_RDONLY, 0700)
	if err != nil {
		fmt.Println("Failed to open signal file: " + err.Error())
		os.Exit(1)
	}
	inotifyCmd := exec.Command("/usr/bin/inotifywait", inotifyArgs...)
	inotifyCmd.Stderr = os.Stderr // Delete this if inotifywait becomes annoying
	for {
		err := inotifyCmd.Run()
		if err != nil {
			fmt.Println("Could not watch signal file: ", err.Error())
			os.Exit(1)
		}
		scanner := bufio.NewScanner(fd)
		args := []string{}
		for scanner.Scan() {
			line := scanner.Text()
			args = append(
				args,
				line,
			)
		}
		go executeAndWait(launchTarget, args)
	}
}

func startMaster(targetExec string, targetArgs []string) {
	startCmd := exec.Command(targetExec, targetArgs...)
	startCmd.Stdin = os.Stdin
	startCmd.Stdout = os.Stdout
	startCmd.Stderr = os.Stderr
	startNotifier <- true
	startCmd.Start()
	systemd.Notify()
	startCmd.Wait()
	startNotifier <- false
	fmt.Println("Main process exited")
}

func main () {
	go startCounter()
	fmt.Println("Starting helper...")

	// This is horrible, but launchTarget may have spaces
	var rawTarget = os.Getenv("launchTarget")
	targetSlice := strings.Split(rawTarget, " ")
	var rawArgs string = os.Getenv("targetArgs")
	argsSlice := strings.Split(rawArgs, "\n")
	targetArgs := targetSlice[1:]
	targetArgs = append(
		targetArgs,
		argsSlice...
	)
	go auxStart(targetSlice[0])
	if os.Getenv("_portableDebug") == "1" {
		targetSlice[0] = "/usr/bin/bash"
		newArgSlice := []string{
			"--noprofile",
			"--rcfile", "/run/bashrc",
		}
		targetArgs = append(
			newArgSlice,
			targetArgs...
		)
	} else if os.Getenv("_portableBusActivate") == "1" {
		rawBus := os.Getenv("busLaunchTarget")
		if len(rawBus) > 0 {
			busTarget := strings.Split(rawBus, " ")
			busArg := busTarget[1:]
			targetSlice[0] = busTarget[0]
			newArgs := []string{}
			newArgs = append(
				newArgs,
				busArg...
			)
		} else {
			fmt.Println("Undefined busLaunchTarget!")
		}
	}
	startMaster(targetSlice[0], targetArgs)
}