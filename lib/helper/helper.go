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
	"syscall"
	"golang.org/x/sys/unix"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/rymdport/portal/notification"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
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
	terminateNotify		= make(chan int, 1)
	procAttr		= &syscall.SysProcAttr{
		Pdeathsig:	syscall.SIGKILL,
	}
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
			terminateNotify <- 1
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
	if len(files) == 0 {
		return origin
	}
	replacerPairs := make([]string, 0 , len(files) * 2)
	for key, val := range files {
		replacerPairs = append(replacerPairs, key, val)
	}
	replacer := strings.NewReplacer(replacerPairs...)
	var result = make([]string, 0, len(origin))
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
	cmd.SysProcAttr = procAttr
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
	os.MkdirAll("/run/portable-control", 0700)
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

func sendPidFd() {
	pid := os.Getpid()
	var st unix.Stat_t

	pidfd, err := unix.PidfdOpen(pid, 0)
	if err != nil {
		fmt.Println("Could not obtain PIDFD: " + err.Error())
		return
	}
	err = unix.Fstat(pidfd, &st)
	if err != nil {
		fmt.Println("Could not obtain PIDFD inode: " + err.Error())
		return
	}
	res, err := daemon.SdNotify(false, "MAINPIDFDID=" + strconv.Itoa(int(st.Ino)))
	if err != nil {
		fmt.Println("Could not set Main PID: " + err.Error())
	} else if res == false {
		fmt.Println("Could not set Main PID: " + "unknown error")
	}
}

func terminateWatcher(blocker chan int, conn *dbus.Conn) {
	busName := "top.kimiblock.portable." + os.Getenv("appID")
	busObj := conn.Object(busName, "/top/kimiblock/portable/daemon")
	<- blocker
	fmt.Println("Requesting termination...")
	call := busObj.Call("top.kimiblock.Portable.Controller.Stop", dbus.FlagNoReplyExpected)
	if call.Err != nil {
		panic(call.Err)
	}
	os.Exit(0)
}

type busStartProcessor struct{
	cmdPfx		[]string
}

func (m *busStartProcessor) AuxStart (
	customTgt bool, tray bool, customExec []string, args []string,
	) (
	hasFd bool,
	stdin dbus.UnixFDIndex,
	stdout dbus.UnixFDIndex,
	stderr dbus.UnixFDIndex,
	busErr *dbus.Error,
	) {
		if tray {
			fmt.Println("Tray activation not supported yet")
			return
		}
		var cmdline []string
		if customTgt {
			fmt.Println("Custom launchTarget not supported yet")
			return
		} else {
			cmdline = m.cmdPfx
		}

		cmdline = append(cmdline, args...)
		fmt.Println("Received start request from D-Bus:", cmdline)
		hasFd = true
		cmd := exec.Command(cmdline[0], cmdline[1:]...)
		cmd.SysProcAttr = procAttr
		tmpIn, err := os.CreateTemp("", "stdin-*")
		if err != nil {
			fmt.Println("Could not create temporary file: " + err.Error())
			return
		}
		defer os.Remove(tmpIn.Name())
		tmpOut, err := os.CreateTemp("", "stdout-*")
		if err != nil {
			fmt.Println("Could not create temporary file: " + err.Error())
			return
		}
		defer os.Remove(tmpOut.Name())
		tmpErr, err := os.CreateTemp("", "stderr-*")
		if err != nil {
			fmt.Println("Could not create temporary file: " + err.Error())
			return
		}
		defer os.Remove(tmpErr.Name())
		cmd.Stdin = tmpIn
		cmd.Stdout = tmpOut
		cmd.Stderr = tmpErr
		return
	}

func busAuxStart(conn *dbus.Conn, cmdPfx []string) {
	proc := new(busStartProcessor)
	proc.cmdPfx = cmdPfx
	var objPath = "/top/kimiblock/portable/init"
	var busName = os.Getenv("appID") + ".Portable.Helper"

	err := conn.Export(proc, dbus.ObjectPath(objPath), "top.kimiblock.Portable.Init")
	if err != nil {
		panic(err)
	}

	node := &introspect.Node{
		Interfaces:	[]introspect.Interface{
			{
				Name:		"top.kimiblock.Portable.Init",
				Methods:	[]introspect.Method{
					{
						Name:		"AuxStart",
						Args:		[]introspect.Arg{
							{
								Name:		"CustomTarget",
								Type:		"b",
								Direction:	"in",
							},
							{
								Name:		"TrayActivate",
								Type:		"b",
								Direction:	"in",
							},
							{
								Name:		"TargetExec",
								Type:		"as",
								Direction:	"in",
							},
							{
								Name:		"Args",
								Type:		"as",
								Direction:	"in",
							},
							{
								Name:		"HasFileDescriptors",
								Type:		"b",
								Direction:	"out",
							},
							{
								Name:		"stdin",
								Type:		"h",
								Direction:	"out",
							},
							{
								Name:		"stdout",
								Type:		"h",
								Direction:	"out",
							},
							{
								Name:		"stderr",
								Type:		"h",
								Direction:	"out",
							},
						},
					},
				},
			},
		},
	}
	conn.Export(introspect.NewIntrospectable(node), dbus.ObjectPath(objPath), "org.freedesktop.DBus.Introspectable")
	reply, err := conn.RequestName(busName, dbus.NameFlagDoNotQueue)
	if err != nil {
		panic(err)
	}
	switch reply {
		case dbus.RequestNameReplyPrimaryOwner:
			fmt.Println("Successfully owned bus name")
		default:
			fmt.Println("Could not own bus name: " + reply.String())
			os.Exit(1)
	}

}

type AuxStartMsg struct {
	CustomTarget	bool
	TargetExec	[]string
	Args		[]string
	ID		int
}

func busSigListener(sig chan *dbus.Signal, cmdPfx []string) {
	for signal := range sig {
		switch signal.Name {
			case "top.kimiblock.Portable.Controller" + ".AuxStart":
				var msg AuxStartMsg
				err := dbus.Store(signal.Body, &msg)
				if err != nil {
					fmt.Println("Could not decode AuxStart broadcast: " + err.Error())
					return
				}
				fmt.Println("Received AuxStart broadcast from D-Bus:", msg)
				var cmdline []string
				if msg.CustomTarget {
					cmdline = msg.TargetExec
				} else {
					cmdline = cmdPfx
				}
				cmdline = append(cmdline, msg.Args...)
				cmd := exec.Command(cmdline[0], cmdline[1:]...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.SysProcAttr = procAttr
				cmd.Start()
				startNotifier <- true
				cmd.Wait()
				startNotifier <- false
				// TODO: support FD store
		}
	}
}

func main () {
	var busWg sync.WaitGroup
	var bus *dbus.Conn
	busWg.Go(func() {
		var err error
		bus, err = dbus.ConnectSessionBus()
		if err != nil {
			panic("Could not connect to session bus: " + err.Error())
		}
		fmt.Println("Connected to session bus")
	})
	go startCounter()
	go sendPidFd()

	// This is horrible, but launchTarget may have spaces
	var rawTarget = os.Getenv("launchTarget")
	targetSlice := strings.Split(rawTarget, " ")
	targetArgs := targetSlice[1:]
	targetArgs = append(
		targetArgs,
		os.Args[1:]...
	)
	fmt.Println("Got raw command line arguments:", targetArgs)
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
	busWg.Wait()
	go busAuxStart(bus, targetSlice)
	go startMaster(targetSlice[0], args)
	go terminateWatcher(terminateNotify, bus)
	select {}
}