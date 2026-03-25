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

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/godbus/dbus/v5"
	"github.com/landlock-lsm/go-landlock/landlock"
	landlockSyscall "github.com/landlock-lsm/go-landlock/landlock/syscall"
)

type PassFiles struct {
	// FileMap is a map that contains [host path string](docid string)
	FileMap		map[string]string
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

func engageLandlock () {
	config, err := landlock.NewConfig(landlock.ScopedSet(landlockSyscall.ScopeSignal))
	if err != nil {
		fmt.Println("Could not restrict sending signals: " + err.Error())
	} else {
		config.RestrictScoped()
	}

	mountInfoCfg := os.Getenv("_portableNoFlatpakInfo")
	if len(mountInfoCfg) > 0 {
		return
	} else {
		fullAccRule := landlock.AccessFSSet(landlockSyscall.AccessFSExecute|landlockSyscall.AccessFSWriteFile|landlockSyscall.AccessFSReadFile|
landlockSyscall.AccessFSReadDir|landlockSyscall.AccessFSRemoveDir|landlockSyscall.AccessFSRemoveFile|landlockSyscall.AccessFSMakeChar|landlockSyscall.AccessFSMakeDir|landlockSyscall.AccessFSMakeReg|landlockSyscall.AccessFSMakeSock|landlockSyscall.AccessFSMakeFifo|landlockSyscall.AccessFSMakeBlock|landlockSyscall.AccessFSMakeSym|landlockSyscall.AccessFSRefer|landlockSyscall.AccessFSTruncate|landlockSyscall.AccessFSIoctlDev)
		dirAccRule := landlock.AccessFSSet(landlockSyscall.AccessFSExecute|landlockSyscall.AccessFSWriteFile|landlockSyscall.AccessFSReadFile|
landlockSyscall.AccessFSReadDir|landlockSyscall.AccessFSRemoveDir|landlockSyscall.AccessFSRemoveFile|landlockSyscall.AccessFSMakeDir|landlockSyscall.AccessFSMakeReg|landlockSyscall.AccessFSMakeSock|landlockSyscall.AccessFSMakeFifo|landlockSyscall.AccessFSMakeSym|landlockSyscall.AccessFSRefer|landlockSyscall.AccessFSTruncate)
		dirRoRule := landlock.AccessFSSet(landlockSyscall.AccessFSExecute|landlockSyscall.AccessFSReadFile|
landlockSyscall.AccessFSReadDir)
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("Could not get user home: " + err.Error())
			return
		}
		err = landlock.V6.RestrictPaths(
			landlock.PathAccess(landlockSyscall.AccessFSReadDir, "/"), // Root
			landlock.PathAccess(dirRoRule, "/bin"),
			landlock.PathAccess(fullAccRule, "/dev"),
			landlock.PathAccess(fullAccRule, "/proc"),
			landlock.PathAccess(fullAccRule, "/sys"),
			landlock.PathAccess(dirRoRule, "/etc"),
			landlock.PathAccess(dirRoRule, "/lib"),
			landlock.PathAccess(dirRoRule, "/lib64"),
			landlock.PathAccess(dirRoRule, "/opt"),
			landlock.PathAccess(dirRoRule, "/sbin"),
			landlock.PathAccess(dirRoRule, "/usr"),
			landlock.ROFiles("/.flatpak-info"),
			landlock.PathAccess(dirAccRule, "/run"),
			landlock.PathAccess(dirAccRule, "/tmp"),
			landlock.PathAccess(dirAccRule, homeDir),
		)
		if err != nil {
			fmt.Println("Could not enforce file system landlock: " + err.Error())
		}
	}
}

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
func startCounter () {
	var countLock sync.RWMutex
	var startedCount int
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
				incoming.cmd.Stdout = os.Stdout
				incoming.cmd.Stderr = os.Stderr
				incoming.cmd.Stdin = os.Stdin
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

func startMaster(targetExec []string, targetArgs []string) {
	var startReq StartNofifyMsg
	rawEnv := os.Getenv("_portableHelperExtraFiles")
	var args []string
	if len(rawEnv) > 0 {
		var decoded PassFiles
		err := json.Unmarshal([]byte(rawEnv), &decoded)
		if err != nil {
			panic("Could not decode JSON from environment variable: " + err.Error())
		}
		fmt.Println("Replacing cmdline using file map:", decoded)

		targetExec = cmdlineReplacer(targetExec, decoded.FileMap)
		targetArgs = cmdlineReplacer(targetArgs, decoded.FileMap)
	}
	args = append(targetExec, targetArgs...)

	startCmd := exec.Command(args[0], args[1:]...)
	fmt.Println("Starting main application", args[0], "with cmdline:", args[1:])
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

func main () {
	var landlockWg sync.WaitGroup
	landlockWg.Go(func() {
		engageLandlock()
	})
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
	var rawTarget = os.Getenv("_portableLaunchTarget")
	var targetSlice = []string{
		rawTarget,
	}
	targetArgs := os.Args[1:]
	fmt.Println("Got raw command line arguments:", targetArgs)
	exposedEnvs := os.Getenv("_portableHelperExtraFiles")
	if os.Getenv("_portableDebug") == "1" {
		targetSlice = []string{
			"/usr/bin/bash",
			"--noprofile",
			"--rcfile", "/run/bashrc",
		}
	} else if os.Getenv("_portableBusActivate") == "1" {
		busArgs := []string{}
		err := json.Unmarshal([]byte(os.Getenv("_portableBusActivateArgs")), &busArgs)
		if err != nil {
			panic(err)
		}
		if len(os.Args) > 1 {
			targetArgs = os.Args[1:]
		} else {
			targetArgs = []string{}
		}
		targetSlice = busArgs
	} else if len(exposedEnvs) > 0 {
		var exposeList PassFiles
		json.Unmarshal([]byte(exposedEnvs), &exposeList)
	}
	busWg.Wait()
	landlockWg.Wait()
	go busAuxStart(bus, targetSlice)
	go startMaster(targetSlice, targetArgs)
	go terminateWatcher(terminateNotify, bus)
	select {}
}