package main

import (
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
	fmt.Println("Updating tracking status: ", count)
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

func handleIncomingAuxConn(conn net.Conn, launchTarget string, launchArgs []string) {
	ioRead, err := io.ReadAll(conn)
	if err != nil {
		fmt.Println("Could not read connection: " + err.Error())
		return
	}
	rawCmdline := strings.TrimRight(string(ioRead), "\n")
	targetArgs := []string{}
	targetArgs = append(
		targetArgs,
		launchArgs...
	)
	decodedArgs := []string{}
	err = json.Unmarshal([]byte(rawCmdline), &targetArgs)
	if err != nil {
		fmt.Println("Could not unmarshal cmdline: " + err.Error())
		return
	}
	targetArgs = append(
		targetArgs,
		decodedArgs...
	)
	go executeAndWait(launchTarget, targetArgs)
}

func auxStart (launchTarget string, launchArgs []string) {
	var signalSocket string = "/run/portable-control/auxStart"
	os.RemoveAll(signalSocket)
	socket, err := net.Listen("unix", signalSocket)
	if err != nil {
		fmt.Println("Could not listen for aux start: " + err.Error())
		return
	}
	defer socket.Close()
	var connCount int
	for {
		conn, connErr := socket.Accept()
		connCount++
		fmt.Println("Handling aux signal, total count: ", connCount)
		if connErr != nil {
			fmt.Println("Could not listen for aux start: " + connErr.Error())
		}
		go handleIncomingAuxConn(conn, launchTarget, launchArgs)
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