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

func statusNotifier(count chan int) {
	for {
		stringNum := strconv.Itoa(<-count)
		fmt.Println("Updating tracking status: " + stringNum)
		systemd.UpdateStatus("Tracking processes: " + stringNum)
	}
}

func startCounter () {
	var startedCount int = 0
	var notifierChan = make(chan int, 2147483647)
	go statusNotifier(notifierChan)
	for {
		incoming := <- startNotifier
		if incoming == true {
			startedCount++
		} else {
			startedCount--
		}

		notifierChan <- startedCount

		if startedCount == 0 {
			fmt.Println("All tracked processes have exited")
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
	fmt.Println("Main process exited")
	startNotifier <- false
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
	startMaster(targetSlice[0], targetArgs)
}