package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	godbus "github.com/godbus/dbus/v5"
)

func terminateInstance(config Config) {
	conn, err := godbus.SessionBus()
	busName := "top.kimiblock.portable." + config.Metadata.AppID
	if err != nil {
		pecho("crit", "Could not connect to session bus:", err)
	}
	busObj := conn.Object(busName, "/top/kimiblock/portable/daemon")
	call := busObj.Call("top.kimiblock.Portable.Controller.Stop", godbus.FlagNoReplyExpected)
	if call.Err != nil {
		pecho("warn", "Could not terminate instance: " + call.Err.Error())
	} else {
		pecho("debug", "Requested termination via D-Bus")
	}
	os.Exit(0)
}

func wakeInstance(config Config, docMap chan PassFiles) {
	conn, err := godbus.SessionBus()
	if err != nil {
		pecho("crit", "Could not connect to session bus:", err)
	}

	if config.Advanced.TrayWake {
		err := trayWakeNG(config, conn)
		if err != nil {
			pecho("crit", "Could not wake remote: " + err.Error())
		}
	} else {
		busAuxStartReq(conn, false, runtimeOpt.applicationArgs, config, docMap)
	}
}


type startReply struct {
	hasDescriptors	bool
	baseDir		string
}

func busAuxStartReq(conn *godbus.Conn, tray bool, args []string, config Config, docMap chan PassFiles) {
	//oldIn := os.Stdin
	//oldOut := os.Stdout
	//oldErr := os.Stderr

	var files PassFiles

	files = <- docMap

	var fileSlice []string

	for key, val := range files.FileMap {
		fileSlice = append(fileSlice, key, val)
	}

	busObj := conn.Object(config.Metadata.AppID + ".Portable.Helper", "/top/kimiblock/portable/init")
	var sl []string
	var customTgt bool
	if runtimeOpt.isDebug {
		sl = []string{
			"/usr/bin/bash",
			"--noprofile",
			"--rcfile", "/run/bashrc",
			"-i",
		}
		args = []string{}
		customTgt = true
	}

	var counter int
	var call *godbus.Call

	for {
		if counter > 10 {
			pecho("crit", "Failed to wake existing instance")
			break
		}
		counter++
		call = busObj.Call("top.kimiblock.Portable.Init.AuxStart", godbus.FlagNoAutoStart,
			customTgt,
			tray,
			sl,
			args,
			fileSlice,
		)
		if call.Err != nil {
			pecho("warn", "Could not emit start signal: " + call.Err.Error())
		} else {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}


	var reply startReply
	err := call.Store(&reply.hasDescriptors, &reply.baseDir)
	if err != nil {
		pecho("crit", "Could not decode bus reply: " + err.Error())
	}
	if reply.hasDescriptors == false {
		pecho("debug", "Remote has no descriptors, returning...")
		return
	}
	fmt.Println("Streaming console from sandbox, press Control-D to detach")
	baseDir := reply.baseDir
	inFile := filepath.Join(baseDir, "stdin")
	outFile := filepath.Join(baseDir, "stdout")
	errFile := filepath.Join(baseDir, "stderr")

	var wg sync.WaitGroup
	wg.Go(func() {
		conn, err := net.Dial("unix", outFile)
		if err != nil {
			pecho("warn", "Could not stream standard output: " + err.Error())
			return
		} else {
			defer conn.Close()
			pecho("debug", "Streaming standard output")
		}
		n, err := io.Copy(os.Stdout, conn)
		if err != nil {
			pecho("warn", "Stream finished with error: " + err.Error())
		}
		pecho("debug", "Streamed stdout: " + strconv.Itoa(int(n)) + " bytes")
	})
	wg.Go(func() {
		conn, err := net.Dial("unix", errFile)
		if err != nil {
			pecho("warn", "Could not stream standard error: " + err.Error())
			return
		} else {
			defer conn.Close()
		}
		n, err := io.Copy(os.Stderr, conn)
		if err != nil {
			pecho("warn", "Stream finished with error: " + err.Error())
		}
		pecho("debug", "Streamed stderr: " + strconv.Itoa(int(n)) + " bytes")
	})
	go func() {
		conn, err := net.Dial("unix", inFile)
		if err != nil {
			pecho("warn", "Could not stream standard input: " + err.Error())
			return
		} else {
			defer conn.Close()
		}
		n, err := io.Copy(conn, os.Stdin)
		if err != nil {
			pecho("warn", "Stream finished with error: " + err.Error())
		}
		pecho("debug", "Streamed stdin: " + strconv.Itoa(int(n)) + " bytes")
	} ()
	wg.Wait()

}