package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"net"
	"github.com/rclone/rclone/lib/systemd"
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
		if incoming == true {
			startedCount++
		} else {
			startedCount = startedCount - 1
		}

		go updateSd(startedCount)

		if startedCount < 1 {
			fmt.Println("All tracked processes have exited")
			text := []string{"terminate-now"}
			sendSignal(text)
			fmt.Println("Sent termination signal")
			break
		}
	}
}

func executeAndWait (launchTarget string, args []string) {
	cmd := exec.Command(launchTarget, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println("Executing auxiliary target: ", launchTarget + " with " + strings.Join(args, " "))
	fmt.Println("Argument count: ", len(args))
	cmd.Start()
	startNotifier <- true
	cmd.Wait()
	startNotifier <- false
}

func auxStart (launchTarget string, launchArgs []string) {
	inotifyArgs := []string{
		"--quiet",
		"-e",
		"close_write",
		"/run/startSignal",
	}
	// var previousSig string
	for {
		inotifyCmd := exec.Command("/usr/bin/inotifywait", inotifyArgs...)
		inotifyCmd.Stderr = os.Stderr // Delete this if inotifywait becomes annoying
		errInotify := inotifyCmd.Run()
		inotifyCmd.Stdout = io.Discard
		time.Sleep(50 * time.Millisecond)
		// TODO: use UNIX socket for signalling
		if errInotify != nil {
			fmt.Println("Could not watch signal file: ", errInotify.Error())
			os.Exit(1)
		}
		fd, err := os.OpenFile("/run/startSignal", os.O_RDONLY, 0700)
		if err != nil {
			fmt.Println("Failed to open signal content: " + err.Error())
			os.Exit(1)
		}
		scanner := bufio.NewScanner(fd)
		var args string
		var index int = 1
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) == 0 {
				continue
			} else if line == "false" && index == 1 {
				continue
			}
			args = args + line
			index++
		}
		fmt.Println("Got raw argument line: " + args)
		targetArgs := []string{}
		extArgs := []string{}
		json.Unmarshal([]byte(args), &extArgs)
		targetArgs = append(
			launchArgs,
			extArgs...
		)
		go executeAndWait(launchTarget, targetArgs)
		fd.Close()
		fd, _ = os.OpenFile("/run/startSignal", os.O_WRONLY|os.O_TRUNC, 0700)
		var content string = ""
		fmt.Fprint(fd, content)
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

func sendSignal(signal []string) {
	socket, err := net.Dial("unix", "/run/portable-control/daemon")
	if err != nil {
		panic("Could not dial signal socket" + err.Error())
	}
	var finalSignal string
	for _, value := range signal {
		finalSignal = value + "\n"
	}
	_, errWrite := socket.Write([]byte(finalSignal))
	if errWrite != nil {
		panic("Could not write signal " + finalSignal + ": " + errWrite.Error())
	}
}

func main () {
	go startCounter()
	fmt.Println("Starting helper...")

	// This is horrible, but launchTarget may have spaces
	var rawTarget = os.Getenv("launchTarget")
	targetSlice := strings.Split(rawTarget, " ")
	//var rawArgs string = os.Getenv("targetArgs")
	var rawArgs = []string{}
	json.Unmarshal([]byte(os.Getenv("targetArgs")), &rawArgs)
	fmt.Println("Got raw command line arguments: " + strings.Join(rawArgs, " "))
	targetArgs := targetSlice[1:]
	targetArgs = append(
		targetArgs,
		rawArgs...
	)
	go auxStart(targetSlice[0], targetSlice[1:])
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