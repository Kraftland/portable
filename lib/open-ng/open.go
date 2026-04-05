package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/KarpelesLab/reflink"
	"github.com/godbus/dbus/v5"
)

var (
	linkRegexp	=	regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.-]*://[^\s/$.?#].[^\s]*`);
)

func openPath(path string, showItem bool) {
	modPath, _ := evalPath(path)

	log.Println("evalPath returned path", modPath)

	if len(modPath) == 0 {
		log.Fatalln("Failed to resolve path")
		return
	}

	stat, err := os.Stat(modPath)
	if err != nil {
		log.Fatalln("Could not stat path: " + err.Error())
		return
	}

	isDir := stat.IsDir()

	if showItem == false {
		succ := openPathPortal(modPath, isDir)
		if succ == true {
			return
		}
	}

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Fatalln("Could not connect to session bus: " + err.Error())
		return
	}
	defer conn.Close()
	log.Println("Calling FileManager1 for path: " + modPath)
	pathSlice := []string{"file://" + modPath}
	fileManager1Obj := conn.Object("org.freedesktop.FileManager1", "/org/freedesktop/FileManager1")
	fileManager1Obj.Call(
		"org.freedesktop.FileManager1.ShowItems",
		0,
		pathSlice,
		os.Getenv("appID"),
	)
}

func evalPath(path string) (finalPath string, modified bool) {
	inputAbs, err := filepath.Abs(path)
	if err != nil {
		log.Fatalln("Could not get absolute path: " + err.Error())
		return
	}

	inputAbs, _ = strings.CutPrefix(path, "file://")

	log.Println("Resolved absolute path", inputAbs)


	sandboxHome, err := filepath.Abs(os.Getenv("HOME"))
	if err != nil {
		log.Fatalln("Could not get home path: " + err.Error())
		return
	}

	var userName string
	userInfo, err := user.Current()
	if err != nil {
		log.Println("Could not get current user name")
	} else {
		userName = userInfo.Username
	}

	if inputAbs == sandboxHome {
		finalPath = sandboxHome
		return
	} else if strings.HasPrefix(inputAbs, filepath.Join("/home", userName)) {
		if strings.Contains(inputAbs, sandboxHome) == false {
			finalPath = sandboxHome
			return
		}
	} else if inputAbs == "/home" {
		finalPath = sandboxHome
		return
	}
	if strings.HasPrefix(inputAbs, sandboxHome) {
		modified = false
		finalPath = inputAbs
		log.Println("Translated sandbox path " + path + " to " + finalPath)
		return
	}

	openBlacklist := []string{
		"/sandbox",
		"/.flatpak-info",
		"/run",
		"/media",
		"/mnt",
		"/proc",
		"/root",
		"/srv",
		"/tmp",
		"top.kimiblock.portable",
		"/var",
		filepath.Join(sandboxHome, "options"),
		filepath.Join(sandboxHome, ".flatpak-info"),
		filepath.Join(sandboxHome, ".var"),
	}

	sharedPath := filepath.Join(
		sandboxHome,
		"Shared",
		filepath.Base(inputAbs),
	)

	for _, val := range openBlacklist {
		if strings.HasPrefix(inputAbs, val) {
			modified = true
			log.Println("Rewriting path")
			os.RemoveAll(sharedPath)
			err := reflink.Auto(inputAbs, sharedPath)
			if err != nil {
				log.Fatalln("Could not copy shared file: " + err.Error())
				return
			}
			finalPath = sharedPath
			break
		}
	}

	if modified == false {
		log.Println("Linking unknown path")
		os.RemoveAll(sharedPath)
		err := os.Symlink(inputAbs, sharedPath)
		if err != nil {
			log.Fatalln("Could not link path: " + err.Error())
		}
		finalPath = filepath.Dir(sharedPath)
	}

	log.Println("Translated " + path + " to " + finalPath)
	return
}

func openPathPortal(path string, dir bool) (success bool) {
	busConn, err := dbus.ConnectSessionBus()
	if err != nil {
		panic(err)
	}
	busname := busConn.Names()[0]
	absName := strings.ReplaceAll(strings.TrimPrefix(busname, ":"), ".", "_")
	var resp = make(chan uint32, 1)
	inId := "portableOpen" + strconv.Itoa(rand.Int())
	var objPath string = filepath.Join("/org/freedesktop/portal/desktop/request", absName, inId)

	fd, err := os.Open(path)
	if err != nil {
		log.Fatalln("Could not open path:", err)
	}
	var wg sync.WaitGroup
	wg.Add(1)

	go func () {
		err := busConn.AddMatchSignal(
			dbus.WithMatchInterface("org.freedesktop.portal.Request"),
			dbus.WithMatchMember("Response"),
			dbus.WithMatchObjectPath(dbus.ObjectPath(objPath)),
			dbus.WithMatchSender("org.freedesktop.portal.Desktop"),
		)
		if err != nil {
			panic(err)
		}
		sigChan := make(chan *dbus.Signal, 512)
		busConn.Signal(sigChan)
		wg.Done()
		for sig := range sigChan {
			log.Println("Incoming signal")
			var v dbus.Variant
			var stat uint32
			if sig.Path == dbus.ObjectPath(objPath) && sig.Name == "org.freedesktop.portal.Request.Response" {
				err := dbus.Store(sig.Body, &stat, &v)
				if err != nil {
					log.Fatalln("Could not store bus reply:", err)
				}
				resp <- stat
				break
			}
		}
		defer busConn.RemoveSignal(sigChan)
	} ()

	const parentWindow string = ""
	fdBus := dbus.UnixFD(fd.Fd())
	var optMap = make(map[string]dbus.Variant)
	optMap["handle_token"] = dbus.MakeVariant(inId)
	optMap["writable"] = dbus.MakeVariant(true)
	optMap["ask"] = dbus.MakeVariant(true)
	var portalObj dbus.BusObject
	portalObj = busConn.Object(
		"org.freedesktop.portal.Desktop",
		"/org/freedesktop/portal/desktop",
	)
	var busMethod string
	switch dir {
		case true:
			busMethod = "OpenDirectory"
		case false:
			busMethod = "OpenFile"
	}
	wg.Wait()
	call := portalObj.Call(
		"org.freedesktop.portal.OpenURI."+ busMethod,
		0,
		parentWindow,
		fdBus,
		optMap,
	)
	if call.Err != nil {
		log.Println("Call to Portal failed:", call.Err)
		return false
	} else {
		log.Println("Called Portal:", call.Body)
	}
	res := <- resp
	log.Println("Got response from Portal:", res)
	switch res {
		case 0:
			os.Exit(0)
			return true
		case 1:
			log.Println("Interaction cancelled")
			os.Exit(0)
			return true
		case 2:
			log.Println("User interaction was ended in some other way")
			os.Exit(0)
			return true
		default:
			log.Println("Unexpected Response signal:", res)
			return false
	}
}

func main () {
	rawCmdArgs := os.Args
	fmt.Println("Received command line open request: " + strings.Join(rawCmdArgs, ", "))
	if os.Args[1] == "--show-item" {
		fmt.Println("Activating legacy dde-file-manager mode")
		totalLength := len(os.Args)
		var loopCounter uint = 2
		for {
			if loopCounter > uint(totalLength) {
				fmt.Println("Could not resolve path")
				break
			}
			_, err := os.Stat(os.Args[loopCounter])
			if err != nil {
				fmt.Println("Invalid argument: " + os.Args[loopCounter])
			} else {
				openPath(os.Args[loopCounter], true)
				break
			}
			loopCounter++
		}
	} else if strings.Contains(os.Args[1], "file://") == false && linkRegexp.Match([]byte(os.Args[1])) {
		fmt.Println("Got a link")
		err := OpenURI(os.Args[1])
		if err != nil {
			warn.Fatalln("Could not open link:", err)
		}
	} else {
		openPath(os.Args[1], false)
	}
}