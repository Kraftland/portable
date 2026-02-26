package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/rymdport/portal/notification"
)

type PassFiles struct {
	// FileMap is a map that contains [host path string](docid string)
	FileMap		map[string]string
}

type pipeInfo struct {
	cmdline			[]string
	id			int
	stdin			io.WriteCloser
	stdout			io.ReadCloser
	stderr			io.ReadCloser
}

type ResponseField struct {
	Success			bool
	ID			int
}

var (
	startNotifier		= make(chan bool, 32)
	// Should check for collision first!
	pipeMapGlob		= make(map[int]pipeInfo)
	pipeLock		sync.RWMutex
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

func cmdlineReplacer(origin []string, files map[string]string) []string {
	replacerPairs := make([]string, 0 , len(files) * 2)
	for key, val := range files {
		replacerPairs = append(replacerPairs, key, val)
	}
	replacer := strings.NewReplacer(replacerPairs...)
	var result []string
	for _, val := range origin {
		result = append(result, replacer.Replace(val))
	}
	return result
}

func getIdFromReq(req *http.Request) (id int, res bool)  {
	header := req.Header
	idRaw := header["Portable"]
	id, err := strconv.Atoi(idRaw[0])
	if err != nil {
		fmt.Println("Could not get request ID: " + err.Error())
		return
	}
	res = true
	return
}

func stdinPipeHandler (writer http.ResponseWriter, req *http.Request) {
	flusher, _ := writer.(http.Flusher)
	defer req.Body.Close()
	id, res := getIdFromReq(req)
	if res == false {
		fmt.Println("Could not handle stdin pipe request")
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	pipeLock.RLock()
	info := pipeMapGlob[id]
	pipeLock.RUnlock()
	fmt.Println("Handling request ID: " + strconv.Itoa(id), "with proto", req.Proto)
	writer.WriteHeader(http.StatusOK)
	flusher.Flush()
	if info.stdin == nil {
		fmt.Println("Could not pipe terminal: I/O nil")
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type","application/octet-stream")
	flusher.Flush()
	_, err := io.Copy(info.stdin, req.Body)
	if err != nil {
		writer.WriteHeader(http.StatusGone)
		fmt.Println("Could not stream stdin: " + err.Error())
		flusher.Flush()
	}
	// Optional: accept a JSON first using bufio
}

func stdoutPipeHandler (writer http.ResponseWriter, req *http.Request) {
	pipeR, pipeW := io.Pipe()
	flusher, _ := writer.(http.Flusher)
	defer req.Body.Close()
	id, res := getIdFromReq(req)
	if res == false {
		fmt.Println("Could not handle stdout pipe request")
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	pipeLock.RLock()
	info := pipeMapGlob[id]
	pipeLock.RUnlock()
	fmt.Println("Handling request ID: " + strconv.Itoa(id), "with proto", req.Proto)
	//writer.WriteHeader(http.StatusOK)
	//flusher.Flush()
	if info.stdout == nil {
		fmt.Println("Could not pipe terminal: I/O nil")
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type","application/octet-stream")
	writer.WriteHeader(http.StatusOK)
	flusher.Flush()
	//mw := io.MultiWriter(os.Stdout, pipeW)
	const newLine = "\n"
	go func() {
		scanner := bufio.NewScanner(pipeR)
		for scanner.Scan() {
			writer.Write(scanner.Bytes())
			writer.Write([]byte(newLine))
			flusher.Flush()
		}
	} ()
	_, err := io.Copy(pipeW, info.stdout)
	if err != nil {
		writer.WriteHeader(http.StatusGone)
		fmt.Println("Could not stream stdout: " + err.Error())
		flusher.Flush()
	}
	// Optional: accept a JSON first using bufio
}

func stderrPipeHandler (writer http.ResponseWriter, req *http.Request) {
	pipeR, pipeW := io.Pipe()
	flusher, _ := writer.(http.Flusher)
	defer req.Body.Close()
	id, res := getIdFromReq(req)
	if res == false {
		fmt.Println("Could not handle stderr pipe request")
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	pipeLock.RLock()
	info := pipeMapGlob[id]
	pipeLock.RUnlock()
	fmt.Println("Handling request ID: " + strconv.Itoa(id), "with proto", req.Proto)
	//writer.WriteHeader(http.StatusOK)
	//flusher.Flush()
	if info.stderr == nil {
		var cycleCounter int
		var fail bool
		for {
			if cycleCounter > 5 {
				fail = true
				break
			}
			cycleCounter++
			time.Sleep(1 * time.Second)
		}
		if fail {
			fmt.Println("Could not pipe terminal: I/O nil")
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	writer.Header().Set("Content-Type","application/octet-stream")
	writer.WriteHeader(http.StatusOK)
	flusher.Flush()
	//mw := io.MultiWriter(os.Stderr, writer)
	const newLine = "\n"
	go func() {
		scanner := bufio.NewScanner(pipeR)
		for scanner.Scan() {
			writer.Write(scanner.Bytes())
			writer.Write([]byte(newLine))
			flusher.Flush()
		}
	} ()
	_, err := io.Copy(pipeW, info.stderr)
	if err != nil {
		writer.WriteHeader(http.StatusGone)
		fmt.Println("Could not stream stderr: " + err.Error())
		flusher.Flush()
	}
	// Optional: accept a JSON first using bufio
}

func auxStartHandler (writer http.ResponseWriter, req *http.Request) {
	fmt.Println("Handling aux start request")
	var reqDecode StartRequest
	bodyRaw, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Println("Could not read request: " + err.Error())
		return
	}
	defer req.Body.Close()
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
	filesMap := reqDecode.Files.FileMap
	fmt.Println("Got file map from request:", filesMap)
	cmdlineNew := cmdlineReplacer(cmdline, filesMap)
	cmd := exec.Command(cmdlineNew[0], cmdlineNew[1:]...)
	//cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL
	var id int
	for {
		id = rand.Int()
		pipeLock.RLock()
		_, ok := pipeMapGlob[id]
		pipeLock.RUnlock()
		if ok == false {
			fmt.Println("Selected ID", id)
			break
		}
	}
	var resp ResponseField
	resp.ID = id
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Could not pipe standard output", err)
		jsonObj, _ := json.Marshal(resp)
		writer.Write(jsonObj)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println("Could not pipe standard error", err)
		jsonObj, _ := json.Marshal(resp)
		writer.Write(jsonObj)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("Could not pipe standard input", err)
		jsonObj, _ := json.Marshal(resp)
		writer.Write(jsonObj)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	var pipeInf pipeInfo
	pipeInf.cmdline = cmdlineNew
	pipeInf.id = id
	pipeInf.stderr = stderrPipe
	pipeInf.stdin = stdinPipe
	pipeInf.stdout = stdoutPipe

	fmt.Println("Executing command:", cmdline)
	err = cmd.Start()
	if err != nil {
		fmt.Println("Could not start command: ", err)
		jsonObj, _ := json.Marshal(resp)
		writer.Write(jsonObj)
		writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	resp.Success = true
	jsonObj, _ := json.Marshal(resp)
	writer.Write(jsonObj)

	startNotifier <- true
	go procWatcher(cmd, id)
	pipeLock.Lock()
	pipeMapGlob[id] = pipeInf
	pipeLock.Unlock()

	//maps.stderr.Close()
	//maps.stdin.Close()
	//maps.stdout.Close()
}

func procWatcher (cmd *exec.Cmd, id int) {
	err := cmd.Wait()
	if err != nil {
		fmt.Println("Command returned error: ", err)
	}
	startNotifier <- false
	pipeLock.Lock()
	maps := pipeMapGlob[id]
	maps.id = 0
	pipeMapGlob[id] = maps
	pipeLock.Unlock()
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
	mux.HandleFunc("/stream/stdin", stdinPipeHandler)
	mux.HandleFunc("/stream/stdout", stdoutPipeHandler)
	mux.HandleFunc("/stream/stderr", stderrPipeHandler)
	h2s := &http2.Server{}
	h2cMux := h2c.NewHandler(mux, h2s)
	server := &http.Server {
		Handler:	h2cMux,
		ConnContext:	func(ctx context.Context, c net.Conn) context.Context {
			cmdPrefix := []string{launchTarget}
			cmdPrefix = append(cmdPrefix, launchArgs...)
			return context.WithValue(ctx, "cmdPrefix", cmdPrefix)
		},
	}
	err = http2.ConfigureServer(server, h2s)
	if err != nil {
		panic(err)
	}
	err = server.Serve(socketHttp)
	if err != nil {
		panic(err)
	}
}

func startMaster(targetExec string, targetArgs []string) {
	rawEnv := os.Getenv("_portableHelperExtraFiles")
	if len(rawEnv) > 0 {
		var decoded PassFiles
		err := json.Unmarshal([]byte(rawEnv), &decoded)
		if err != nil {
			panic("Could not decode JSON from environment variable: " + err.Error())
		}
		fmt.Println("Replacing cmdline using file map:", decoded)
		var execSlice = []string{targetExec}
		targetExec = cmdlineReplacer(execSlice, decoded.FileMap)[0]
		targetArgs = cmdlineReplacer(targetArgs, decoded.FileMap)
	}
	startCmd := exec.Command(targetExec, targetArgs...)
	startCmd.Stdin = os.Stdin
	startCmd.Stdout = os.Stdout
	startCmd.Stderr = os.Stderr
	startNotifier <- true
	fmt.Println("Starting main application", targetExec, "with cmdline:", targetArgs)
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
		time.Sleep(360000 * time.Hour)
	}
}