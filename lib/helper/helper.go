package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"golang.org/x/sys/unix"
	"net"
	"path/filepath"
	"math/rand"

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

type StartNofifyMsg struct {
	cmd			*exec.Cmd
	UDS			[]net.Listener

}

var (
	startNotifier		= make(chan StartNofifyMsg, 32)
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
	var countLock sync.RWMutex
	var startedCount int = 0
	fmt.Println("Start counter init done")
	for incoming := range startNotifier {
		go func() {
			var stopStreaming = make(chan int, 1)
			var blockWg sync.WaitGroup
			if len(incoming.UDS) == 3 {
				blockWg.Add(3)
				go func () {
					if incoming.UDS[0] == nil {
						fmt.Println("Could not stream: nil socket")
						return
					}
					conn, err := incoming.UDS[0].Accept()
					if err != nil {
						fmt.Println("Could not accept connection:", err)
						return
					}
					go func () {
						<- stopStreaming
						conn.Close()
					} ()
					inP, err := incoming.cmd.StdinPipe()
					blockWg.Done()
					if err != nil {
						fmt.Println("Could not accept connection:", err)
						return
					}
					n, err := io.Copy(inP, conn)
					fmt.Println("Streamed", n, "bytes of stdin")
				} ()
				go func () {
					if incoming.UDS[1] == nil {
						fmt.Println("Could not stream: nil socket")
						return
					}
					conn, err := incoming.UDS[1].Accept()
					if err != nil {
						fmt.Println("Could not accept connection:", err)
						return
					}
					defer conn.Close()
					pipe, err := incoming.cmd.StdoutPipe()
					blockWg.Done()
					if err != nil {
						fmt.Println("Could not accept connection:", err)
						return
					}
					n, err := io.Copy(conn, pipe)
					fmt.Println("Streamed", n, "bytes of stdout")
				} ()
				go func () {
					if incoming.UDS[2] == nil {
						fmt.Println("Could not stream: nil socket")
						return
					}
					conn, err := incoming.UDS[2].Accept()
					if err != nil {
						fmt.Println("Could not accept connection:", err)
						return
					}
					defer conn.Close()
					pipe, err := incoming.cmd.StderrPipe()
					blockWg.Done()
					if err != nil {
						fmt.Println("Could not accept connection:", err)
						return
					}
					n, err := io.Copy(conn, pipe)
					fmt.Println("Streamed", n, "bytes of stderr")
				} ()
			} else {
				fmt.Println("Not piping console: Listeners mismatch")
			}
			blockWg.Wait()
			err := incoming.cmd.Start()
			if err != nil {
				fmt.Println("Could not start executable with:", incoming.cmd.Args, err)
				return
			}

			countLock.Lock()
			startedCount++
			go updateSd(startedCount)
			countLock.Unlock()
			err = incoming.cmd.Wait()
			go func () {
				stopStreaming <- 1
				for _, val := range incoming.UDS {
					val.Close()
				}
			} ()

			if err != nil {
				fmt.Println("Command with argument: ", incoming.cmd.Args, "failed:", err)
			}
			countLock.Lock()
			startedCount--
			go updateSd(startedCount)
			countLock.Unlock()
			countLock.RLock()
			if startedCount == 0 {
				daemon.SdNotify(false, daemon.SdNotifyStopping)
				fmt.Println("All tracked processes have exited")
				terminateNotify <- 1
				return
			}
			countLock.RUnlock()
		} ()
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

func startMaster(targetExec string, targetArgs []string) {
	var startReq StartNofifyMsg
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
	fmt.Println("Starting main application", targetExec, "with cmdline:", targetArgs)
	go daemon.SdNotify(false, daemon.SdNotifyReady)
	startReq.cmd = startCmd
	startNotifier <- startReq
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
	isStream bool,
	baseDir	string,
	busErr *dbus.Error,
	) {
		path := os.Getenv("XDG_RUNTIME_DIR")
		if len(path) == 0 {
			fmt.Println("XDG_RUNTIME_DIR not set")
			return
		}
		var req StartNofifyMsg
		if tray {
			fmt.Println("Tray activation not supported yet")
			return
		}
		var cmdline []string
		if customTgt {
			cmdline = customExec
		} else {
			cmdline = m.cmdPfx
		}

		var sockDir string
		var trials int

		for {
			if trials > 512 {
				fmt.Println("Could not pick temp dir")
				return
			}
			trials++
			id := rand.Int()
			idCand := strconv.Itoa(id)
			sockDir = filepath.Join(path, "portable", os.Getenv("appID"), "stream", idCand)

			_, err := os.Stat(sockDir)
			if err != nil {
				err := os.MkdirAll(sockDir, 0700)
				if err != nil {
					fmt.Println("Could not create directory for stream: " + err.Error())
				} else {
					break
				}
			} else {
				continue
			}
		}

		baseDir = sockDir

		inAddr, err := net.ResolveUnixAddr("unix", filepath.Join(sockDir, "stdin"))
		if err != nil {
			fmt.Println("Could not resolve address: " + err.Error())
			return
		}
		stdinListen, err := net.ListenUnix("unix", inAddr)
		if err != nil {
			fmt.Println("Could not stream command:", err)
			return
		}
		outAddr, err := net.ResolveUnixAddr("unix", filepath.Join(sockDir, "stdout"))
		if err != nil {
			fmt.Println("Could not resolve address: " + err.Error())
			return
		}
		stdoutListen, err := net.ListenUnix("unix", outAddr)
		if err != nil {
			fmt.Println("Could not stream command:", err)
			return
		}
		errAddr, err := net.ResolveUnixAddr("unix", filepath.Join(sockDir, "stderr"))
		if err != nil {
			fmt.Println("Could not resolve address: " + err.Error())
			return
		}
		stderrListen, err := net.ListenUnix("unix", errAddr)
		if err != nil {
			fmt.Println("Could not stream command:", err)
			return
		}

		cmdline = append(cmdline, args...)
		fmt.Println("Received start request from D-Bus:", cmdline)
		cmd := exec.Command(cmdline[0], cmdline[1:]...)
		cmd.SysProcAttr = procAttr
		isStream = true
		req.cmd = cmd
		req.UDS = []net.Listener{stdinListen, stdoutListen, stderrListen}
		startNotifier <- req
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
								Name:		"IsStream",
								Type:		"b",
								Direction:	"out",
							},
							{
								Name:		"BaseDir",
								Type:		"s",
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