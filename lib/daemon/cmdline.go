package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	//"github.com/Kraftland/portable/lib/portals"
	"github.com/godbus/dbus/v5"
	godbus "github.com/godbus/dbus/v5"
)

func openHome (config Config) {
	cmdArgs := []string{
		filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory),
	}
	openCmd := exec.Command("/usr/bin/xdg-open", cmdArgs...)
	openCmd.Stderr = os.Stderr
	openCmd.Run()
	os.Exit(0)
}

func resetDocs (config Config) {
	cmdArgs := []string{
		"permission-reset",
		config.Metadata.AppID,
	}
	openCmd := exec.Command("flatpak", cmdArgs...)
	openCmd.Stderr = os.Stderr
	openCmd.Run()
	os.Exit(0)
}


func cmdlineDispatcher(cmdChan chan int8, config Config, exposeChan chan map[string]string) {
	var skipCount	int
	var hasExpose	bool
	var fileFwd	bool
	var wg		sync.WaitGroup
	var exposeMap = map[string]string{}
	cmdlineArray := os.Args
	runtimeOpt.applicationArgs = config.Exec.Arguments
	for index, value := range cmdlineArray {
		if index == 0 {
			continue
		}
		if skipCount > 0 {
			skipCount--
			continue
		} else if runtimeOpt.argStop == true {
			runtimeOpt.applicationArgs = append(
				runtimeOpt.applicationArgs,
				value,
			)
			continue
		}
		switch value {
			case "--file-forwarding", "--forward-file":
				pecho("debug", "File forwarding enabled via:", value)
				fileFwd = true
			case "--expose":
				if len(cmdlineArray) <= index + 2 {
					pecho("warn", "--expose requires 2 arguments")
					break
				}
				if filepath.IsAbs(cmdlineArray[index + 1]) {
					pecho("debug", "Validated absolute path")
				} else {
					pecho("warn", "Rejecting non absolute path")
					continue
				}
				hasExpose = true
				skipCount += 2
				exposeMap[cmdlineArray[index + 1]] = cmdlineArray[index + 2]
			case "--actions" :
			skipCount++
			if len(cmdlineArray) <= index + 1 {
				pecho("warn", "--actions requires an argument")
				break
			}
			switch cmdlineArray[index + 1] {
				case "quit":
					pecho("debug", "Received quit request from user")
					terminateInstance(config)
					os.Exit(0)
				case "debug-shell":
					addEnv("_portableDebug=1")
					runtimeOpt.isDebug = true
				case "share-file", "share-files":
					err := shareFileViaHelper(config, false)
					if err != nil {
						pecho("warn", "Unable to request file sharing via IPC, falling back:", err)
						err := alertHelperNotRunning(config)
						if err != nil {
							pecho(
								"warn",
								"Could not send notification:",
								err,
							)
						}
					}
					abortChan <- true
				case "share-directories", "share-directory":
					err := shareFileViaHelper(config, true)
					if err != nil {
						pecho("warn", "Unable to request directory sharing via IPC:", err)
						err := alertHelperNotRunning(config)
						if err != nil {
							pecho(
								"warn",
								"Could not send notification:",
								err,
							)
						}
					}
					abortChan <- true
				case "opendir", "home", "openhome":
					openHome(config)
					abortChan <- true
				case "reset-document", "reset-documents":
					resetDocs(config)
					abortChan <- true
				case "stat", "stats":
					showStats(config)
					abortChan <- true
				case "f5aaebc6-0014-4d30-beba-72bce57e0650":
					pecho("warn", "Portable has removed the ability to start in unsafe mode, please use the legacy version instead")
					abortChan <- true
				default:
					pecho("warn", "Unrecognised action: " + cmdlineArray[index + 1])
			}
			case "--dbus-activation":
				addEnv("_portableBusActivate=1")
				if ! config.BusActivation.Enable {
					pecho("crit", "Could not start application: bus activation not enabled")
				}
			case "--":
				runtimeOpt.argStop = true
			case "--help":
				printHelp()
				pecho("crit", "Aborted start to print help")
			default:
				pecho("warn", "Unrecognised option: " + value)
		}
	}
	wg.Go(func() {
		if ! hasExpose {
			return
		}
		exposeList := []string{}
		for key := range exposeMap {
			exposeList = append(exposeList, key)
		}
		exposeChan <- exposeMap
	})
	wg.Go(func() {
		var mp = make(map[string]string)
		if ! fileFwd {
			return
		}
		for _, val := range runtimeOpt.applicationArgs {
			if slices.Contains(config.Exec.Arguments, val) {
				pecho("debug", "Skipping forwarding of configuration defined argument", val)
				continue
			}
			if ! filepath.IsAbs(val) {
				pecho("warn", "Rejecting non-absolute forwarding option:", val)
				continue
			}
			mp[val] = "null"
		}
		exposeChan <- mp
	})
	wg.Go(func() {
		encodedArg, errEncode := json.Marshal(runtimeOpt.applicationArgs)
		if errEncode != nil {
			pecho("warn", "Could not encode arguments as json:", errEncode)
			return
		}
		addEnv("targetArgs=" + string(encodedArg))
	})
	wg.Wait()
	cmdChan <- 1
	pecho("debug", "Full command line:", cmdlineArray)
	pecho("info", "Application arguments:", runtimeOpt.applicationArgs)
}

func shareFileViaHelper(config Config, directory bool) error {
	conn, err := godbus.SessionBus()
	if err != nil {
		return err
	}
	busObj := conn.Object(
		config.Metadata.AppID + ".Portable.Helper",
		"/top/kimiblock/portable/init",
	)
	call := busObj.Call(
		"top.kimiblock.Portable.Init.RequestFSAccess",
		godbus.FlagAllowInteractiveAuthorization,
		directory,
	)
	if call.Err != nil {
		return call.Err
	}
	return nil
}

func alertHelperNotRunning(config Config) error {
	conn, err := godbus.SessionBus()
	if err != nil {
		return err
	}
	obj := conn.Object(
		"org.freedesktop.Notifications",
		"/org/freedesktop/Notifications",
	)

	call := obj.Call(
		"org.freedesktop.Notifications.Notify",
		0,
		"Portable Daemon",
		uint32(0),
		"dialog-warning-symbolic",
		config.Metadata.FriendlyName + " not running or responding",
		"Sharing objects needs the app to be running.",
		[]string{},
		make(map[string]dbus.Variant),
		int32(7),
	)
	if call.Err != nil {
		return call.Err
	}
	return nil
}

func showStats(config Config) {
	conn, err := godbus.ConnectSessionBus()
	busName := "top.kimiblock.portable." + config.Metadata.AppID
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	busObj := conn.Object(busName, "/top/kimiblock/portable/daemon")

	size := getDirSize(filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory))
	var builder strings.Builder
	builder.WriteString("Application Statistics: \n")
	builder.WriteString("	Total disk use: " + strconv.FormatFloat(size,'f', 2, 64) + "M\n")
	call := busObj.Call(busName + ".Ping", 0)
	if call.Err != nil {
		pecho("debug", "Could not call running instance")
	} else {
		pecho("debug", "Remote instance responded with Pong")
		call = busObj.Call("top.kimiblock.portable.Info.GetInfo", 0)
		if call.Err != nil {
			pecho(
				"warn",
				"Could not obtain runtime info from remote: " + call.Err.Error(),
			)
		} else {
			var reply []string
			err := call.Store(&reply)
			if err != nil {
				pecho("warn", "Could not decode remote reply: " + err.Error())
			} else {
				builder.WriteString("Runtime Status: \n")
				for _, val := range reply {
					builder.WriteString("	" + val + "\n")
				}
			}
		}
	}

	fmt.Println(builder.String())
	os.Exit(0)
}

func getDirSize(path string) float64 {
	sizeChan := make(chan int, 512)
	var wg sync.WaitGroup
	var totalBytes int
	var totalFiles int
	wg.Go(func() {
		for size := range sizeChan {
			totalBytes = totalBytes + size
			totalFiles++
		}
	})
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		stat, errStat := os.Stat(path)
		if errStat != nil {
			pecho("debug", "Could not stat " + path + ": " + errStat.Error())
			return nil
		}
		sizeChan <- int(stat.Size())
	return nil
	})
	close(sizeChan)
	if err != nil {
		return 0
	}
	wg.Wait()
	pecho("debug", "Calculated " + strconv.Itoa(totalFiles) + " files")
	return bytesToMb(totalBytes)
}
