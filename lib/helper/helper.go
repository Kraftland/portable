package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/rymdport/portal/notification"
)

type PassFiles struct {
	// FileMap is a map that contains [host path string](docid string)
	FileMap		map[string]string
}

var (
	startNotifier		= make(chan bool, 32767)
)

func updateSd(count int) {
	status := "STATUS=" + "Tracking processes: " + strconv.Itoa(count)
	sent, err := daemon.SdNotify(false, status)
	if sent == true {
		fmt.Println("Updated tracking status: ", count)
	} else {
		if err == nil {
			fmt.Println("Notification of daemon status not supported")
		} else {
			fmt.Println("Failed to update status: " + err.Error())
		}
	}
}

func netsockFailNotification() {
	failmsg := os.Getenv("netsockFail")
	if len(failmsg) > 0 {
		var content = notification.Content {
			Title:		"Failed to apply firewall",
			Body:		failmsg,
			Priority:	"urgent",
		}
		err := notification.Add(2147483647, content)
		if err != nil {
			fmt.Println("Failed to send notification: " + err.Error())
		}
	}
}

func startCounter () {
	go netsockFailNotification()
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
			daemon.SdNotify(false, daemon.SdNotifyStopping)
			fmt.Println("All tracked processes have exited")
			text := []string{"terminate-now"}
			sendSignal(text)
			fmt.Println("Sent termination signal")
			break
		}
	}
}
type StartRequest struct {
	Exec		[]string
	CustomTarget	bool
	Files		PassFiles
}

func auxStartHandler (writer http.ResponseWriter, req *http.Request) {
	fmt.Println("Handling aux start request")
	var reqDecode StartRequest
	bodyRaw, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Println("Could not read request: " + err.Error())
		return
	}
	err = json.Unmarshal(bodyRaw, &reqDecode)
	if err != nil {
		fmt.Println("Could not decode request: " + err.Error())
		return
	}
	cmdPfx := req.Context().Value("cmdPrefix").([]string)
	var cmdline []string
	if reqDecode.CustomTarget {
		cmdline = reqDecode.Exec
	} else {
		cmdline = append(cmdPfx, reqDecode.Exec...)
	}

	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println("Executing command:", cmdline)
	startNotifier <- true
	req.Body.Close()
	cmd.Run()
	startNotifier <- false
}

func auxStart (launchTarget string, launchArgs []string) {
	var httpSockPath string = "/run/portable-control/helper"
	socketHttp, err := net.Listen("unix", httpSockPath)
	if err != nil {
		fmt.Println("Could not listen on helper socket: " + err.Error())
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/start", auxStartHandler)
	server := &http.Server {
		Handler:	mux,
		ConnContext:	func(ctx context.Context, c net.Conn) context.Context {
			cmdPrefix := []string{launchTarget}
			cmdPrefix = append(cmdPrefix, launchArgs...)
			return context.WithValue(ctx, "cmdPrefix", cmdPrefix)
		},
	}
	server.Serve(socketHttp)
}

func startMaster(targetExec string, targetArgs []string) {
	startCmd := exec.Command(targetExec, targetArgs...)
	startCmd.Stdin = os.Stdin
	startCmd.Stdout = os.Stdout
	startCmd.Stderr = os.Stderr
	startNotifier <- true
	fmt.Println("Starting main application " + targetExec + " with cmdline: " + strings.Join(targetArgs, " "))
	startCmd.Start()
	daemon.SdNotify(false, daemon.SdNotifyReady)
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
	exposedEnvs := os.Getenv("_portableHelperExtraFiles")
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
	} else if len(exposedEnvs) > 0 {
		var exposeList PassFiles
		json.Unmarshal([]byte(exposedEnvs), &exposeList)
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