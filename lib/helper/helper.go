package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"bufio"
	"github.com/rclone/rclone/lib/systemd"
	"time"
)

var (
	startNotifier		= make(chan bool, 32767)
)

func updateSd(count int) {
	fmt.Println("Updating signal: ", count)
	systemd.UpdateStatus("Tracking processes: " + strconv.Itoa(count))
}

func startCounter () {
	var startedCount int = 0
	fmt.Println("Start counter init done")
	for {
		incoming := <- startNotifier
		fmt.Println("Got signal: ", incoming)
		if incoming == true {
			startedCount++
		} else {
			startedCount = startedCount - 1
		}

		go updateSd(startedCount)

		if startedCount < 1 {
			fmt.Println("All tracked processes have exited")
			const text = "terminate-now"
			fd, err := os.OpenFile("/run/startSignal", os.O_WRONLY|os.O_TRUNC, 0700)
			if err != nil {
				fmt.Println("Unable to open signal file: " + err.Error())
			}
			fmt.Fprintln(fd, text)
			fd.Close()
			fmt.Println("Sent termination signal")
			break
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
	for {
		fd, err := os.OpenFile("/run/startSignal", os.O_RDONLY, 0700)
		if err != nil {
			fmt.Println("Failed to open signal file: " + err.Error())
			os.Exit(1)
		}
		inotifyCmd := exec.Command("/usr/bin/inotifywait", inotifyArgs...)
		inotifyCmd.Stderr = os.Stderr // Delete this if inotifywait becomes annoying
		errInotify := inotifyCmd.Run()
		if errInotify != nil {
			fmt.Println("Could not watch signal file: ", err.Error())
			os.Exit(1)
		}
		scanner := bufio.NewScanner(fd)
		args := []string{}
		var index int = 1
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) == 0 {
				continue
			} else if line == "false" && index == 1 {
				continue
			}
			args = append(
				args,
				line,
			)
			index++
		}
		go executeAndWait(launchTarget, args)
		fd.Close()
	}
}

func startMaster(targetExec string, targetArgs []string) {
	startCmd := exec.Command(targetExec, targetArgs...)
	startCmd.Stdin = os.Stdin
	startCmd.Stdout = os.Stdout
	startCmd.Stderr = os.Stderr
	startNotifier <- true
	fmt.Println("Starting main application " + targetExec + " with cmdline: " + strings.Join(targetArgs, " "))
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
	fmt.Println("Got raw command line arguments: " + rawArgs)
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
	args := []string{}
	for _, arg := range targetArgs {
		if len(arg) > 0 {
			args = append(
				args,
				arg,
			)
		}
	}
	startMaster(targetSlice[0], args)
	for {
		time.Sleep(360000 * time.Second)
	}
}