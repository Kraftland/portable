package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func setCBreak(file *os.File) (func () (), error) {
	termios, err := unix.IoctlGetTermios(int(file.Fd()), unix.TCGETS)
	if err != nil {
		return nil, err
	}
	oldState := termios
	termios.Lflag &^= unix.ICANON
	termios.Lflag &^= unix.ECHO

	err = unix.IoctlSetTermios(int(file.Fd()), unix.TCSETS, termios)
	if err != nil {
		return nil, err
	}
	cancelFunc := func () {
		unix.IoctlSetTermios(int(file.Fd()), unix.TCSETS, oldState)
	}
	return cancelFunc, nil
}

// Converts the current terminal (stdin) into a raw console, caller should call cancelfunc when done
func rawTerm() (func () (), error) {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}
	return func() {
		term.Restore(int(os.Stdin.Fd()), oldState)
	}, nil
}

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

	var env_map = make(map[string]string)
	if val := os.Getenv("XDG_ACTIVATION_TOKEN"); len(val) > 0 {
		env_map["XDG_ACTIVATION_TOKEN"] = val
	}

	if config.Advanced.TrayWake {
		err := trayWakeNG(config, conn)
		if err != nil {
			pecho("crit", "Could not wake remote: " + err.Error())
		}
	} else {
		ver, err := getHelperVersion(conn, config)
		if err != nil {
			pecho("warn", "Could not get helper version:", err)
		}
		switch ver {
			case 18:
				busAuxStartV18(
					conn,
					runtimeOpt.applicationArgs,
					config,
					docMap,
					env_map,
				)
			default:
				busAuxStartReq(
					conn,
					false,
					runtimeOpt.applicationArgs,
					config,
					docMap,
				)
		}
	}
}


// Gets the API version for Helper, and wait for it to come online of not
func getHelperVersion(conn *godbus.Conn, config Config) (uint, error) {
	var helper_name = config.Metadata.AppID + ".Portable.Helper"

	var counter uint = 0

	WaitLoop:
	for {
		if counter > 100 {
			return 0, errors.New("Could not call helper: maximum retry reached")
		}
		obj := conn.Object(
			"org.freedesktop.DBus",
			"/org/freedesktop/DBus",
		)
		call := obj.Call(
			"org.freedesktop.DBus.NameHasOwner",
			godbus.FlagNoAutoStart,
			helper_name,
		)
		if call.Err != nil {
			pecho("warn", "Could not call messaging bus:", call.Err)
			return 0, errors.New("Could not query helper state: " + call.Err.Error())
		}
		var active bool
		err := call.Store(&active)
		if err != nil {
			return 0, errors.New("Could not query helper state: " + err.Error())
		}
		switch active {
			case true:
				break WaitLoop
			case false:
				time.Sleep(100 * time.Millisecond)
				counter++
		}
	}

	busObj := conn.Object(
		helper_name,
		"/top/kimiblock/portable/init",
	)
	call := busObj.Call(
		"org.freedesktop.DBus.Properties.Get",
		0,
		"top.kimiblock.Portable.Init",
		"Version",
	)
	if call.Err != nil {
		return 0, call.Err
	}

	var version uint
	err := call.Store(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

type startReply struct {
	hasDescriptors	bool
	baseDir		string
}

func busAuxStartV18(
	conn *godbus.Conn,
	args []string,
	config Config,
	docMap chan PassFiles,
	envs map[string]string,
) {
	restoreTerm, err := rawTerm()
	if err != nil {
		pecho("warn", "Could not set console to raw mode:", err)
	} else {
		defer restoreTerm()
	}

	var files PassFiles

	files = <- docMap

	busObj := conn.Object(config.Metadata.AppID + ".Portable.Helper", "/top/kimiblock/portable/init")
	var custom_target bool
	var target string
	var arguments []string
	if config.isDebug {
		custom_target = true
		target = "bash"
		arguments = []string{
			"--noprofile",
			"--rcfile", "/run/bashrc",
			"-i",
		}
	} else {
		arguments = args
	}
	call := busObj.Call(
		"top.kimiblock.Portable.Init.AuxStart2",
		godbus.FlagAllowInteractiveAuthorization,
		// Bus arguments below
		custom_target,
		target,
		true, // append mode should be on for now
		arguments,
		files.FileMap,
		envs,
	)
	if call.Err != nil {
		pecho("warn", "Could not send start signal:", call.Err)
		return
	}
	var reply startReply
	err := call.Store(&reply.baseDir)
	if err != nil {
		pecho("crit", "Could not decode bus reply:", err)
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

func busAuxStartReq(conn *godbus.Conn, tray bool, args []string, config Config, docMap chan PassFiles) {
	var files PassFiles

	files = <- docMap

	var fileSlice []string

	for key, val := range files.FileMap {
		fileSlice = append(fileSlice, key, val)
	}

	busObj := conn.Object(config.Metadata.AppID + ".Portable.Helper", "/top/kimiblock/portable/init")
	var sl []string
	var customTgt bool
	if config.isDebug {
		sl = []string{
			"/usr/bin/bash",
			"--noprofile",
			"--rcfile", "/run/bashrc",
			"-i",
		}
		args = []string{}
		customTgt = true
	}
	call := busObj.Call("top.kimiblock.Portable.Init.AuxStart", godbus.FlagAllowInteractiveAuthorization,
		customTgt,
		tray,
		sl,
		args,
		fileSlice,
	)
	if call.Err != nil {
		pecho("warn", "Could not emit start signal: " + call.Err.Error())
		time.Sleep(2 * time.Second)
		return
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
		restoreTerm, err := setCBreak(os.Stdin)
		if err != nil {
			pecho("warn", "Could not set standard input mode: " + err.Error())
		} else {
			defer restoreTerm()
		}

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